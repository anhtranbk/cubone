package cubone

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	MembershipChannel = "membership"
	OnsiteChannel     = "onsite"
)

const MessageTypeAck = "ack"

var (
	ErrInvalidOnsiteMessage = errors.New("invalid onsite message")
	ErrEndpointNotFound     = errors.New("endpoint not found")
	ErrServerInternal       = errors.New("server internal error")
	errInvalidPayload       = errors.New("invalid payload")
	errInvalidMessageType   = errors.New("invalid message type")
)

type unAckMessage struct {
	data      interface{}
	clientId  string
	createdAt time.Time
}

type OnsiteService struct {
	cfg           *Config
	cm            *ClientManager
	pubsub        PubSub
	doneCh        chan struct{}
	unAckMessages map[string]*unAckMessage
	buf           []string
	codec         MessageCodec
}

func NewOnsiteService(cfg Config) *OnsiteService {
	var (
		pubsub PubSub = nil
		err    error
	)

	if cfg.RedisAddr != "" {
		pubsub, _ = NewRedisPubSubFromAddrStr(cfg.RedisAddr)
	} else if cfg.Redis != nil {
		pubsub, err = NewRedisPubSub(cfg.Redis)
	}

	if err != nil {
		log.Fatalw("could not create Redis from config", "cfg", cfg.Redis, "error", err)
		return nil
	}
	if pubsub == nil {
		pubsub = NewFakePubSub()
		log.Info("no real pubsub service configured, use in-memory fake-pubsub instead")
	}

	return &OnsiteService{
		cfg:           &cfg,
		cm:            NewClientManager(cfg),
		pubsub:        pubsub,
		doneCh:        make(chan struct{}, 1),
		unAckMessages: make(map[string]*unAckMessage),
		buf:           make([]string, 8192),
		codec:         NewDefaultCodec(),
	}
}

func (s *OnsiteService) Start() error {
	if err := s.cm.Startup(); err != nil {
		log.Errorf("start client manager error: %v", err)
		return ErrServerInternal
	}

	go s.run()
	log.Info("onsite service started")
	return nil
}

func (s *OnsiteService) Shutdown() error {
	s.doneCh <- struct{}{}
	_ = s.cm.Shutdown()
	_ = s.pubsub.Close()
	return nil
}

func (s *OnsiteService) ConnectClient(clientId string, ws WSConn) error {
	if err := s.cm.Connect(clientId, ws); err != nil {
		log.Errorw("could not connect to client", "clientId", clientId, "err", err)
		return err
	}

	payload, _ := s.codec.Encode(&MembershipMessage{
		ClientId: clientId,
		OwnerId:  s.cm.ID,
	})
	if err := s.pubsub.Publish(MembershipChannel, payload); err != nil {
		log.Errorw("could not publish message to pub/sub handler, maybe the connection lost. "+
			"Refuse to create new WebSocket connection", "clientId", clientId, "err", err)
		_ = s.DisconnectClient(clientId)
		return err
	}
	return nil
}

func (s *OnsiteService) DisconnectClient(clientId string) error {
	err := s.cm.Disconnect(clientId)
	if err != nil {
		log.Errorw("could not disconnect to client", "clientId", clientId)
	}
	return err
}

func (s *OnsiteService) PublishMessage(msg *DeliveryMessage) error {
	if msg == nil || msg.Endpoint == "" {
		log.Warnw("received invalid message", "msg", msg)
		return ErrInvalidOnsiteMessage
	}

	found, err := s.cm.IsActiveClient(msg.Endpoint)
	if err != nil {
		log.Errorf("checking active client error: %v", err)
		return ErrServerInternal
	}
	if !found {
		log.Warnw("endpoint not found", "endpoint", msg.Endpoint)
		return ErrEndpointNotFound
	}

	payload, err := s.codec.Encode(msg)
	if err != nil {
		log.Errorf("could not publish message: %v", err)
		return err
	}

	if err = s.pubsub.Publish(OnsiteChannel, payload); err != nil {
		log.Errorf("could not publish message: %v", err)
		return err
	}

	log.Debugw("published message", "msg", msg)
	return nil
}

func (s *OnsiteService) run() {
	defer func(s *OnsiteService) {
		_ = s.Shutdown()
	}(s)

	wsCh := s.cm.MessageChannel()
	psCh := s.pubsub.Channel()
	ticker := time.NewTicker(s.cfg.MessageRetryInterval)

	for {
		select {
		case <-s.doneCh:
			return
		case req := <-wsCh:
			s.handleWS(req)
		case msg := <-psCh:
			s.handlePubSub(msg)
		case <-ticker.C:
			s.handleResendUnAckMessages()
		}
	}
}

func (s *OnsiteService) handleWS(req *WSClientRequest) {
	var (
		wsMsg WSClientMessage
		err   error = nil
	)
	if err = s.codec.Decode(req.Payload, &wsMsg); err != nil {
		log.Warnw("received malformed request from client", "req", req, "err", err)
		return
	}

	// NOTE: Older clients send ack message in format
	//{
	//	"type": "ack",
	//	"data": {
	//		"id": "<message_id>"
	//    }
	//}
	// but newer clients will use this format (support multiple message type in one unified struct)
	//{
	//	"type": "ack",
	//	"ack": {
	//		"id": "<message_id>"
	//    }
	//}
	if wsMsg.Type == MessageTypeAck {
		var ackMsg *AckMessage = nil
		ackMsg, err = parseAckMessage(&wsMsg)
		if ackMsg != nil {
			s.onAckMessage(req.ClientId, ackMsg)
		}
	} else {
		err = errInvalidMessageType
	}

	if err != nil {
		log.Warnw("could not handle ws message",
			"clientId", req.ClientId,
			"type", wsMsg.Type,
			"payload", string(req.Payload),
			"err", err)
	}
}

func (s *OnsiteService) handlePubSub(msg *PubSubMessage) {
	if msg.Channel == OnsiteChannel {
		// pub-sub messages are only sent by other servers, so we don't need to care about invalid message format here
		s.onDeliveryMessage(parseDeliveryMessage(s.codec, msg.Payload))
	} else if msg.Channel == MembershipChannel {
		s.onMembershipMessage(parseMembershipMessage(s.codec, msg.Payload))
	} else {
		log.Warnw("received message from unsupported pub-sub channel", "channel", msg.Channel)
	}
}

func (s *OnsiteService) onDeliveryMessage(msg *DeliveryMessage) {
	msgId := generateId()
	if err := s.sendMessage(msg.Endpoint, msgId, msg.Data); err != nil {
		log.Errorw("error occurred while handling delivery message", "msg", msg)
		return
	}

	s.unAckMessages[msgId] = &unAckMessage{
		data:      msg.Data,
		clientId:  msg.Endpoint,
		createdAt: time.Now().Add(s.cfg.MessageRetryTimeout),
	}
}

func (s *OnsiteService) onMembershipMessage(msg *MembershipMessage) {
	if msg.OwnerId == s.cm.ID {
		return
	}
	// Mean that this is not server where client active, we should remove (if present) outdated connection
	if s.cm.IsLocalActiveClient(msg.ClientId) {
		log.Debugw("removing outdated client",
			"clientId", msg.ClientId,
			"newOwner", msg.OwnerId,
			"oldOwner", s.cm.ID)
		if err := s.cm.Disconnect(msg.ClientId); err != nil {
			log.Errorw("error occurred while handling membership message", "msg", msg)
		}
	}
}

func (s *OnsiteService) onAckMessage(clientId string, msg *AckMessage) {
	log.Debugw("received ACK message", "msgId", msg.ID, "clientId", clientId)
	if _, found := s.unAckMessages[msg.ID]; found {
		delete(s.unAckMessages, msg.ID)
		log.Debugw("message handled by client", "msgId", msg.ID, "clientId", clientId)
	}
}

func (s *OnsiteService) handleResendUnAckMessages() {
	if len(s.unAckMessages) == 0 {
		return
	}

	now := time.Now()
	for id, msg := range s.unAckMessages {
		if now.Before(msg.createdAt.Add(s.cfg.MessageRetryInterval)) { // prevent messages from being resent too quickly
			continue
		} else if now.After(msg.createdAt.Add(s.cfg.MessageRetryTimeout)) {
			s.buf = append(s.buf, id)
		} else if !s.cm.IsLocalActiveClient(msg.clientId) {
			// could not send message to target client due to client was disconnected
			// we will not discard the message here and try to deliver message at next run
			log.Debug("could not send un-ack message to a disconnect client", "clientId", msg.clientId)
		} else {
			log.Debugw("re-sending un-ack message to client", "msgId", id, "clientId", msg.clientId)
			if err := s.sendMessage(msg.clientId, id, msg.data); err != nil {
				log.Errorf("error while sending un-ack message to client: %v", err)
			}
		}
	}

	for _, id := range s.buf {
		delete(s.unAckMessages, id)
		log.Infow("un-ack message removed due to exceeded timeout", "id", id)
	}
	log.Infof("%d un-ack messages were removed", len(s.buf))
	// clear buffer
	s.buf = s.buf[:0]
}

func (s *OnsiteService) sendMessage(clientId string, msgId string, data interface{}) error {
	payload, err := s.codec.Encode(&WSServerMessage{
		ID:   msgId,
		Data: data,
	})
	if err != nil {
		log.Errorw("could not send message", "clientId", clientId, "err", err)
		return err
	}
	return s.cm.SendMessage(clientId, payload)
}

func generateId() string {
	return uuid.NewString()
}

func parseAckMessage(wsMsg *WSClientMessage) (*AckMessage, error) {
	var (
		ackMsg *AckMessage
		err    error = nil
	)
	if wsMsg.Ack != nil {
		ackMsg = wsMsg.Ack
	} else if wsMsg.Data != nil {
		// parse message from older clients
		ackMsg, err = parseAckMessageV1(&wsMsg.Data)
	} else {
		err = errInvalidPayload
	}

	return ackMsg, err
}

func parseAckMessageV1(v interface{}) (*AckMessage, error) {
	data, ok := v.(map[string]interface{})
	if !ok {
		return nil, errInvalidPayload
	}

	v, found := data["id"]
	if !found {
		return nil, errInvalidPayload
	}

	id, ok := v.(string)
	if !ok {
		return nil, errInvalidPayload
	}

	return &AckMessage{ID: id}, nil
}

func parseDeliveryMessage(codec MessageCodec, data []byte) *DeliveryMessage {
	var msg DeliveryMessage
	// pub-sub messages are only sent by other servers,
	// so we don't need to care about invalid message format here
	_ = codec.Decode(data, &msg)
	return &msg
}

func parseMembershipMessage(codec MessageCodec, data []byte) *MembershipMessage {
	var msg MembershipMessage
	// pub-sub messages are only sent by other servers,
	// so we don't need to care about invalid message format here
	_ = codec.Decode(data, &msg)
	return &msg
}

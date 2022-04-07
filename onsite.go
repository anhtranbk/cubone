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

var (
	ErrInvalidOnsiteMessage = errors.New("invalid onsite message")
	ErrEndpointNotFound     = errors.New("endpoint not found")
	ErrServerInternal       = errors.New("server internal error")
)

type unAckMessage struct {
	data     interface{}
	clientId string
	timeout  time.Time
}

type OnsiteService struct {
	cfg           *Config
	cm            *ClientManager
	publisher     Publisher
	subscriber    Subscriber
	unAckMessages map[string]*unAckMessage
	doneCh        chan struct{}
}

func NewOnsiteServiceFromConfig(cfg Config) *OnsiteService {
	pubsub := NewFakePubSub()
	return &OnsiteService{
		cfg:           &cfg,
		cm:            NewClientManager(cfg),
		publisher:     pubsub,
		subscriber:    pubsub,
		unAckMessages: make(map[string]*unAckMessage),
		doneCh:        make(chan struct{}, 1),
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
	_ = s.publisher.Close()
	_ = s.subscriber.Close()
	return nil
}

func (s *OnsiteService) ConnectClient(clientId string, ws WSConnection) error {
	err := s.cm.Connect(clientId, ws)
	if err != nil {
		log.Errorw("could not connect to client", "id", clientId, "error", err)
		return err
	}
	err = s.publisher.Publish(MembershipChannel, &MembershipMessage{
		ClientId: clientId,
		OwnerId:  s.cm.ID,
	})
	if err != nil {
		log.Errorw("could not publish message to pub/sub handler, maybe the connection lost. "+
			"Refuse to create new WebSocket connection", "id", clientId)
		_ = s.DisconnectClient(clientId)
	}
	return err
}

func (s *OnsiteService) DisconnectClient(clientId string) error {
	err := s.cm.Disconnect(clientId)
	if err != nil {
		log.Errorw("could not disconnect to client", "id", clientId)
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

	if err := s.publisher.Publish(OnsiteChannel, msg); err != nil {
		log.Errorf("could not publish message: %v", err)
		return err
	}

	log.Debugw("published message: %v", msg)
	return nil
}

func (s *OnsiteService) run() {
	defer func(s *OnsiteService) {
		_ = s.Shutdown()
	}(s)

	wsCh := s.cm.MessageChannel()
	psCh := s.subscriber.Channel()
	for {
		select {
		case <-s.doneCh:
			return
		case msg := <-wsCh:
			s.handleWSMessage(msg)
		case msg := <-psCh:
			s.handlePubSubMessage(msg)
		}
	}
}

func (s *OnsiteService) handleWSMessage(msg *WSClientMessage) {
	ackMsg, err := NewAckMessage(msg)
	if err != nil {
		log.Errorf("could not parse ack message: %v", err)
		return
	}
	s.onAckMessage(ackMsg)
}

func (s *OnsiteService) handlePubSubMessage(msg *PubSubMessage) {
	if msg.Channel == OnsiteChannel {
		s.onDeliveryMessage(NewDeliveryMessage(msg))
	} else if msg.Channel == MembershipChannel {
		s.onMembershipMessage(NewMembershipMessage(msg))
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
		data:     msg.Data,
		clientId: msg.Endpoint,
		timeout:  time.Now().Add(s.cfg.MessageRetryMaxTimeout),
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

func (s *OnsiteService) onAckMessage(msg *AckMessage) {
	log.Debugw("received ACK message", "messageId", msg.ID)
	if _, found := s.unAckMessages[msg.ID]; found {
		delete(s.unAckMessages, msg.ID)
		log.Debugw("message handled by client", "messageId", msg.ID)
	}
}

func (s *OnsiteService) sendMessage(clientId string, msgId string, data interface{}) error {
	return s.cm.SendMessage(clientId, toWSServerMessage(msgId, data))
}

func toWSServerMessage(msgId string, data interface{}) *WSServerMessage {
	return &WSServerMessage{
		ID:   msgId,
		Data: data,
	}
}

func generateId() string {
	return uuid.NewString()
}

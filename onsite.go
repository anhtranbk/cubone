package cubone

import (
	"time"
)

const MembershipChannel = "membership"
const OnsiteChannel = "onsite"

type unacknowledgedMessage struct {
	data     interface{}
	clientId string
	timeout  time.Time
}

type onsiteService struct {
	clientManager *ClientManager
	publisher     Publisher
	subscriber    Subscriber
	unAckMessages map[string]*unacknowledgedMessage
	config        *Config
}

func NewOnsiteService(
	clientManager *ClientManager,
	publisher Publisher,
	subscriber Subscriber,
) *onsiteService {
	return &onsiteService{
		clientManager: clientManager,
		publisher:     publisher,
		subscriber:    subscriber,
		unAckMessages: make(map[string]*unacknowledgedMessage),
	}
}

func NewOnsiteServiceFromConfig(config Config) *onsiteService {
	bloomFilter := NewInmemoryBloomFilter()
	fakePubSub := FakePubSub{}

	return &onsiteService{
		clientManager: NewClientManager(config, bloomFilter),
		publisher:     &fakePubSub,
		subscriber:    &fakePubSub,
		unAckMessages: make(map[string]*unacknowledgedMessage),
		config:        &config,
	}
}

func (s *onsiteService) ConnectClient(clientId string, websocket WebSocketConnection) error {
	err := s.clientManager.Connect(clientId, websocket)
	if err != nil {
		return err
	}
	err = s.publisher.Publish(MembershipChannel, &MembershipMessage{
		ClientId: clientId,
		OwnerId:  s.clientManager.ID,
	})
	if err != nil {
		log.Errorw("Could not publish message to pub/sub handler, maybe the connection lost. "+
			"Refuse to create new WebSocket connection", "clientId", clientId)
		_ = s.DisconnectClient(clientId)
	}
	return err
}

func (s *onsiteService) DisconnectClient(clientId string) error {
	err := s.clientManager.Disconnect(clientId)
	if err != nil {
		log.Errorw("Could not disconnect to client", "clientId", clientId)
	}
	return err
}

func (s *onsiteService) ReceivedWSMessage(clientId string) error {
	panic("implement me")
}

func (s *onsiteService) HandlePubSubMessages() error {
	message, err := s.subscriber.GetMessage(100)
	if err != nil {
		return err
	}

	if message.Channel == OnsiteChannel {
		s.onDeliveryMessage(NewDeliveryMessage(message))
	} else if message.Channel == MembershipChannel {
		s.onMembershipMessage(NewMembershipMessage(message))
	} else {
		log.Warnw("Pub-sub message channel is not supported", "channel", message.Channel)
	}
	return nil
}

func (s *onsiteService) PublishMessage(message *DeliveryMessage) error {
	return s.publisher.Publish(OnsiteChannel, message)
}

func (s *onsiteService) IsMembership(clientId string) (bool, error) {
	return s.clientManager.IsActiveClient(clientId)
}

func (s *onsiteService) GetTotalActiveClients() error {
	panic("implement me")
}

func (s *onsiteService) handlePubSubMessage(message *PubSubMessage) {
	if message.Channel == OnsiteChannel {
		s.onDeliveryMessage(NewDeliveryMessage(message))
	} else if message.Channel == MembershipChannel {
		s.onMembershipMessage(NewMembershipMessage(message))
	} else {
		log.Warnw("Pub-sub message channel is not supported", "channel", message.Channel)
	}
}

func (s *onsiteService) onDeliveryMessage(message *DeliveryMessage) {
	messageId := generateMessageId()
	err := s.sendMessage(message.Endpoint, messageId, message.Data)
	if err != nil {
		log.Errorw("Error occurred while handling delivery message", "message", message)
	}
	s.unAckMessages[messageId] = &unacknowledgedMessage{
		data:     message.Data,
		clientId: message.Endpoint,
		timeout:  time.Now().Add(time.Duration(s.config.MessageRetryMaxTimeout)),
	}
}

func (s *onsiteService) onMembershipMessage(message *MembershipMessage) {
	if message.OwnerId == s.clientManager.ID {
		return
	}
	// Mean that this is not server where client be active, we should remove (if present) outdated connection
	if s.clientManager.IsLocalActiveClient(message.ClientId) {
		log.Debugw("Removing outdated client",
			"clientId", message.ClientId, "ownerId", s.clientManager.ID)
		err := s.clientManager.Disconnect(message.ClientId)
		if err != nil {
			log.Errorw("Error occurred while handling membership message", "message", message)
		}
	}
}

func (s *onsiteService) onAckMessage(message *AckMessage) {
	log.Debugw("Received ACK message", "messageId", message.ID)
	if _, found := s.unAckMessages[message.ID]; found {
		delete(s.unAckMessages, message.ID)
		log.Debugw("Message handled by client", "messageId", message.ID)
	}
}

func (s *onsiteService) sendMessage(clientId string, messageId string, data interface{}) error {
	message := getWrappedMessage(messageId, data)
	return s.clientManager.SendMessage(clientId, message)
}

func getWrappedMessage(messageId string, data interface{}) *WSServerMessage {
	return &WSServerMessage{
		ID:   messageId,
		Data: data,
	}
}

func generateMessageId() string {
	panic("implement me")
}

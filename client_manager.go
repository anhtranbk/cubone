package cubone

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/atomic"
)

const bloomFilterKey = "wsClients"

var (
	ClientNotFoundError = errors.New("client not found")
	ClientIdDuplicated  = errors.New("clientId duplicated")
)

type WebSocketConnection interface {
	Close() error

	SendText(data string) error
	Send(data []byte) error

	ReceiveText() (string, error)
	Receive() ([]byte, error)
}

type ClientStatusListener interface {
	OnConnected(clientId string)
	OnDisconnected(clientId string)
	OnError(clientId string)
}

type MessageHandler func(clientId string, data interface{}) error

type Client struct {
	ID     string
	wsConn WebSocketConnection
}

type ClientManager struct {
	ID               string
	config           Config
	clients          map[string]*Client
	numActiveClients *atomic.Int32
	bloomFilter      BloomFilter
	messageHandlers  map[string][]MessageHandler
}

func NewClientManager(config Config, bloomFilter BloomFilter) *ClientManager {
	cm := &ClientManager{
		ID:               uuid.NewString(),
		config:           config,
		clients:          map[string]*Client{},
		numActiveClients: &atomic.Int32{},
		bloomFilter:      bloomFilter,
		messageHandlers:  make(map[string][]MessageHandler),
	}
	cm.numActiveClients.Store(0)
	log.Infow("ClientManager initialized", "id", cm.ID)
	return cm
}

func (cm *ClientManager) GetClientIds() []string {
	keys := make([]string, 0, len(cm.clients))
	for k := range cm.clients {
		keys = append(keys, k)
	}
	return keys
}

func (cm *ClientManager) IsLocalActiveClient(clientId string) bool {
	_, ok := cm.clients[clientId]
	return ok
}

func (cm *ClientManager) IsActiveClient(clientId string) (bool, error) {
	if _, ok := cm.clients[clientId]; ok {
		//do something here
		return true, nil
	}
	return cm.bloomFilter.Exists(bloomFilterKey, clientId)
}

func (cm *ClientManager) Connect(clientId string, wsConn WebSocketConnection) error {
	if _, found := cm.clients[clientId]; found {
		log.Errorw("Client already existed", "clientId", clientId)
		return ClientIdDuplicated
	}
	if err := cm.bloomFilter.Add(bloomFilterKey, clientId); err != nil {
		return err
	}

	cm.clients[clientId] = &Client{ID: clientId, wsConn: wsConn}
	cm.numActiveClients.Inc()
	log.Infow("Client connected", "clientId", clientId)
	return nil
}

func (cm *ClientManager) Disconnect(clientId string) error {
	log.Debugw("Disconnecting client ...", "clientId", clientId)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("Could not disconnect client, client not found", "clientId", clientId)
		return ClientNotFoundError
	}

	err := client.wsConn.Close()
	if err == nil {
		delete(cm.clients, clientId)
		cm.numActiveClients.Dec()
		log.Infow("Client disconnected", "clientId", clientId)
	}
	return err
}

func (cm *ClientManager) SendMessage(clientId string, message interface{}) error {
	log.Debugw("Sending message", "clientId", clientId, "message", message)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("Client was not found on this handler", "clientId", clientId)
		return ClientNotFoundError
	}

	var err error
	if b, ok := message.([]byte); ok {
		err = client.wsConn.Send(b)
	} else if s, ok := message.(string); ok {
		err = client.wsConn.SendText(s)
	} else {
		data, err2 := json.Marshal(message)
		if err2 != nil {
			return err2
		}
		err = client.wsConn.Send(data)
	}

	if err != nil {
		// Normally this error occurs when handler are trying to send data to a closed connection.
		// TODO: We should disconnect and remove this client ?
		log.Errorw("Error occurred while sending message to client", "error", err.Error())
		_ = cm.Disconnect(clientId)
	}
	return err
}

func (cm *ClientManager) Broadcast(message interface{}) error {
	var err error
	for clientId := range cm.clients {
		err = cm.SendMessage(clientId, message)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *ClientManager) ReceiveMessage(clientId string) (*WSClientMessage, error) {
	client, found := cm.clients[clientId]
	if !found {
		return nil, ClientNotFoundError
	}

	bytes, err := client.wsConn.Receive()
	if err != nil {
		return nil, err
	}
	var message WSClientMessage
	err = json.Unmarshal(bytes, &message)
	return &message, err
}

func (cm *ClientManager) OnMessage(msgType string, handler MessageHandler) {
	handlers, found := cm.messageHandlers[msgType]
	if !found {
		handlers = make([]MessageHandler, 8)
	}
	handlers = append(handlers, handler)
	cm.messageHandlers[msgType] = handlers
}

func (cm *ClientManager) handleMessage(clientId string, message *WSClientMessage) {
	log.Debugw("Message received", "clientId", clientId, "message", message)
	handlers, found := cm.messageHandlers[message.Type]
	if !found {
		log.Warnw("No handlers found for current message type", "type", message.Type)
		return
	}
	for _, handle := range handlers {
		_ = handle(clientId, message)
	}
}

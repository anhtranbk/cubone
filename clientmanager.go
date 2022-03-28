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

	Send(data []byte) error
	SendJson(data interface{}) error
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
	ID           string
	cfg          Config
	clients      map[string]*Client
	totalClients *atomic.Int32
	bf           BloomFilter
	msgCh        chan *WSClientMessage
}

func NewClientManager(cfg Config, bf BloomFilter) *ClientManager {
	cm := &ClientManager{
		ID:           uuid.NewString(),
		cfg:          cfg,
		clients:      map[string]*Client{},
		totalClients: &atomic.Int32{},
		bf:           bf,
	}
	cm.totalClients.Store(0)
	log.Infow("clientManager initialized", "id", cm.ID)
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
	_, found := cm.clients[clientId]
	return found
}

func (cm *ClientManager) IsActiveClient(clientId string) (bool, error) {
	if cm.IsLocalActiveClient(clientId) {
		return true, nil
	}
	return cm.bf.Exists(bloomFilterKey, clientId)
}

func (cm *ClientManager) Connect(clientId string, wsConn WebSocketConnection) error {
	if _, found := cm.clients[clientId]; found {
		log.Errorw("client already existed", "clientId", clientId)
		return ClientIdDuplicated
	}
	if err := cm.bf.Add(bloomFilterKey, clientId); err != nil {
		return err
	}

	client := &Client{ID: clientId, wsConn: wsConn}
	cm.clients[clientId] = client
	cm.totalClients.Inc()
	log.Infow("client connected", "clientId", clientId)

	go cm.receiveMessage(client)

	return nil
}

func (cm *ClientManager) Disconnect(clientId string) error {
	log.Debugw("disconnecting client ...", "clientId", clientId)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("could not disconnect client, client not found", "clientId", clientId)
		return ClientNotFoundError
	}

	err := client.wsConn.Close()
	if err == nil {
		delete(cm.clients, clientId)
		cm.totalClients.Dec()
		log.Infow("client disconnected", "clientId", clientId)
	}
	return err
}

func (cm *ClientManager) SendMessage(clientId string, message interface{}) error {
	log.Debugw("sending message", "clientId", clientId, "message", message)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("client was not found", "clientId", clientId)
		return ClientNotFoundError
	}

	var err error
	if b, ok := message.([]byte); ok {
		err = client.wsConn.Send(b)
	} else if s, ok := message.(string); ok {
		err = client.wsConn.Send([]byte(s))
	} else {
		err = client.wsConn.SendJson(b)
	}

	if err != nil {
		// Normally this error occurs when handler are trying to send data to a closed connection.
		// TODO: We should disconnect and remove this client ?
		log.Errorw("error occurred while sending message to client", "error", err.Error())
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

func (cm *ClientManager) MessageChannel() <-chan *WSClientMessage {
	return cm.msgCh
}

func (cm *ClientManager) receiveMessage(client *Client) error {
	defer cm.Disconnect(client.ID)
	for {
		bytes, err := client.wsConn.Receive()
		if err != nil {
			return err
		}

		var message WSClientMessage
		err = json.Unmarshal(bytes, &message)
		if err != nil {
			log.Warnf("received malformed message: %v", err)
			return err
		}

		log.Debugw("received message", "clientId", client.ID, "message", message)
		cm.msgCh <- &message
	}
}

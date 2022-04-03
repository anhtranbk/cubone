package cubone

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/atomic"
)

var (
	ClientNotFoundErr     = errors.New("client not found")
	ClientIdDuplicatedErr = errors.New("clientId duplicated")
	ClientDisconnectedErr = errors.New("client disconnected")
	TimeoutErr            = errors.New("connection timeout")
)

type ClientStatusListener interface {
	OnConnected(clientId string)
	OnDisconnected(clientId string)
	OnError(clientId string)
}

type MessageHandler func(clientId string, data interface{}) error

type connectRequest struct {
	resCh    chan error
	conn     WSConnection
	clientId string
}

type disconnectRequest struct {
	resCh    chan error
	clientId string
}

type ClientManager struct {
	ID           string
	cfg          Config
	clients      map[string]*Client
	totalClients *atomic.Int32
	doneCh       chan struct{}
	clientMsgCh  chan *WSClientMessage
	connectCh    chan *connectRequest
	disconnectCh chan *disconnectRequest
}

func NewClientManager(cfg Config) *ClientManager {
	cm := &ClientManager{
		ID:           uuid.NewString(),
		cfg:          cfg,
		clients:      map[string]*Client{},
		totalClients: atomic.NewInt32(0),
		doneCh:       make(chan struct{}),
		clientMsgCh:  make(chan *WSClientMessage),
		connectCh:    make(chan *connectRequest),
		disconnectCh: make(chan *disconnectRequest),
	}
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
	// In original Python version, we have a bloom filter service
	// (often served by Redis) to check whether the client is exist
	// Since bloom filter is a data structure that is possible to return
	// "false positive" results... TBD.
	return cm.IsLocalActiveClient(clientId), nil
}

func (cm *ClientManager) Startup() error {
	go cm.run()
	return nil
}

func (cm *ClientManager) Shutdown() error {
	cm.doneCh <- struct{}{}
	return nil
}

func (cm *ClientManager) run() {
	defer cm.cleanup()
	for {
		select {
		case <-cm.doneCh:
			break
		case req := <-cm.connectCh:
			req.resCh <- cm.doConnect(req.clientId, req.conn)
		case req := <-cm.disconnectCh:
			req.resCh <- cm.doDisconnect(req.clientId)
		}
	}
}

func (cm *ClientManager) cleanup() {
	for id, client := range cm.clients {
		if err := client.close(); err != nil {
			log.Errorw("clean up client error", "id", id, "error", err)
		}
	}
}

func (cm *ClientManager) Connect(clientId string, ws WSConnection) error {
	req := &connectRequest{
		resCh:    make(chan error),
		conn:     ws,
		clientId: clientId,
	}
	cm.connectCh <- req

	return waitOrTimeout(req.resCh, cm.cfg.ConnectionTimeout)
}

func (cm *ClientManager) doConnect(clientId string, ws WSConnection) error {
	if _, found := cm.clients[clientId]; found {
		log.Errorw("client already existed", "id", clientId)
		return ClientIdDuplicatedErr
	}

	client := NewClient(clientId, ws, cm.clientMsgCh)
	cm.clients[clientId] = client
	cm.totalClients.Inc()
	log.Infow("client connected", "id", clientId)

	return client.startup()
}

func (cm *ClientManager) Disconnect(clientId string) error {
	req := &disconnectRequest{
		resCh:    make(chan error),
		clientId: clientId,
	}
	cm.disconnectCh <- req

	return waitOrTimeout(req.resCh, cm.cfg.ConnectionTimeout)
}

func (cm *ClientManager) doDisconnect(clientId string) error {
	log.Debugw("disconnecting client ...", "id", clientId)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("could not disconnect client, client not found", "id", clientId)
		return ClientNotFoundErr
	}

	err := client.wsConn.Close()
	if err == nil {
		delete(cm.clients, clientId)
		cm.totalClients.Dec()
		log.Infow("client disconnected", "id", clientId)
	}
	return err
}

func (cm *ClientManager) SendMessage(clientId string, message interface{}) error {
	log.Debugw("sending message", "id", clientId, "message", message)
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("client was not found", "id", clientId)
		return ClientNotFoundErr
	}

	var err error
	if client.closed.Load() {
		return ClientDisconnectedErr
	}
	if b, ok := message.([]byte); ok {
		client.write(b)
	} else if s, ok := message.(string); ok {
		client.write([]byte(s))
	} else {
		b, err = json.Marshal(message)
		client.write(b)
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
	return cm.clientMsgCh
}

func waitOrTimeout(ch <-chan error, timeout time.Duration) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return TimeoutErr
	}
}

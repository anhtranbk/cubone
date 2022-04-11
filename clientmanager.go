package cubone

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/atomic"
)

var (
	ErrClientNotFound     = errors.New("client not found")
	ErrClientIdDuplicated = errors.New("clientId duplicated")
	ErrClientDisconnected = errors.New("client disconnected")
	ErrOperationTimeout   = errors.New("operation timeout")
)

type connectRequest struct {
	resCh    chan error
	conn     WSConn
	clientId string
}

type disconnectRequest struct {
	resCh    chan error
	clientId string
}

type ClientManager struct {
	ID            string
	cfg           Config
	clients       map[string]*Client
	totalClients  *atomic.Int32
	doneCh        chan struct{}
	clientMsgCh   chan *WSClientRequest
	connectCh     chan *connectRequest
	disconnectCh  chan *disconnectRequest
	stateChangeCh chan stateChange
	msgCodec      MessageCodec
}

func NewClientManager(cfg Config) *ClientManager {
	cm := &ClientManager{
		ID:            uuid.NewString(),
		cfg:           cfg,
		clients:       map[string]*Client{},
		totalClients:  atomic.NewInt32(0),
		doneCh:        make(chan struct{}, 1),
		clientMsgCh:   make(chan *WSClientRequest, 1024),
		connectCh:     make(chan *connectRequest, 256),
		disconnectCh:  make(chan *disconnectRequest, 256),
		stateChangeCh: make(chan stateChange, 256),
		msgCodec:      NewDefaultCodec(),
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
	// (often served by Redis) to check whether the client is existed
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
			return
		case req := <-cm.connectCh:
			req.resCh <- cm.doConnect(req.clientId, req.conn)
		case req := <-cm.disconnectCh:
			err := cm.doDisconnect(req.clientId)
			if req.resCh != nil {
				req.resCh <- err
			}
		case change := <-cm.stateChangeCh:
			if change.state == Disconnected {
				_ = cm.doDisconnect(change.clientId)
			}
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

func (cm *ClientManager) Connect(clientId string, ws WSConn) error {
	req := &connectRequest{
		resCh:    make(chan error),
		conn:     ws,
		clientId: clientId,
	}
	cm.connectCh <- req

	return waitOrTimeout(req.resCh, cm.cfg.ConnectionTimeout)
}

func (cm *ClientManager) doConnect(clientId string, ws WSConn) error {
	if _, found := cm.clients[clientId]; found {
		log.Errorw("client already existed", "id", clientId)
		return ErrClientIdDuplicated
	}

	client := NewClient(clientId, ws, cm.clientMsgCh, cm.stateChangeCh)
	cm.clients[clientId] = client
	cm.totalClients.Inc()
	log.Infow("client connected", "id", clientId)
	log.Infof("total client: %v", cm.totalClients)

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
	client, found := cm.clients[clientId]
	if !found {
		return ErrClientNotFound
	}

	delete(cm.clients, clientId)
	cm.totalClients.Dec()
	log.Infof("client disconnected: %v", clientId)

	return client.close()
}

func (cm *ClientManager) SendMessage(clientId string, payload []byte) error {
	log.Debugw("sending message to client", "id", clientId, "message", string(payload))
	client, found := cm.clients[clientId]
	if !found {
		log.Warnw("client was not found", "id", clientId)
		return ErrClientNotFound
	}

	if client.isClosed() {
		log.Errorw("could not send message to closed client", "id", clientId)
		return ErrClientDisconnected
	}

	client.write(payload)
	return nil
}

func (cm *ClientManager) Broadcast(payload []byte) error {
	var err error
	for clientId := range cm.clients {
		if err = cm.SendMessage(clientId, payload); err != nil {
			return err
		}
	}
	return nil
}

func (cm *ClientManager) MessageChannel() <-chan *WSClientRequest {
	return cm.clientMsgCh
}

func waitOrTimeout(ch <-chan error, timeout time.Duration) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return ErrOperationTimeout
	}
}

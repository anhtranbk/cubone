package cubone

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
)

type GorillaWSServer struct {
	upgrader *websocket.Upgrader
}

type GorillaWSConnection struct {
	conn *websocket.Conn
}

func NewGorillaWSServer(config GorillaWsConfig) (*GorillaWSServer, error) {
	return &GorillaWSServer{upgrader: &websocket.Upgrader{
		HandshakeTimeout:  0,
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		WriteBufferPool:   nil,
		Subprotocols:      nil,
		Error:             nil,
		CheckOrigin:       nil,
		EnableCompression: false,
	}}, nil
}

func (s *GorillaWSServer) NewConnection(w http.ResponseWriter, r *http.Request) (WebSocketConnection, error) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &GorillaWSConnection{conn: conn}, nil
}

func (c *GorillaWSConnection) Close() error {
	return c.conn.Close()
}

func (c *GorillaWSConnection) SendText(data string) error {
	return c.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

func (c *GorillaWSConnection) Send(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *GorillaWSConnection) ReceiveText() (string, error) {
	messageType, data, err := c.conn.ReadMessage()
	if messageType != websocket.TextMessage {
		return "", errors.New("unsupported message type " + strconv.Itoa(messageType))
	}
	return string(data), err
}

func (c *GorillaWSConnection) Receive() ([]byte, error) {
	messageType, data, err := c.conn.ReadMessage()
	if messageType != websocket.BinaryMessage {
		return nil, errors.New("unsupported message type " + strconv.Itoa(messageType))
	}
	return data, err
}


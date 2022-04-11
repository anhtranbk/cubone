package cubone

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type GorillaWSConn struct {
	conn *websocket.Conn
}

func NewGorillaWSConnFactory(cfg *GorillaWsConfig) WSConnFactory {
	upgrader := &websocket.Upgrader{
		HandshakeTimeout:  0,
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		WriteBufferPool:   nil,
		Subprotocols:      nil,
		Error:             nil,
		CheckOrigin:       nil,
		EnableCompression: false,
	}

	return WSConnFactory(func(w http.ResponseWriter, r *http.Request) (WSConn, error) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return nil, err
		}
		return &GorillaWSConn{conn: conn}, nil
	})
}

func (c *GorillaWSConn) Close() error {
	return c.conn.Close()
}

func (c *GorillaWSConn) SendText(data string) error {
	return c.conn.WriteMessage(websocket.TextMessage, []byte(data))
}

func (c *GorillaWSConn) Send(data []byte) error {
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *GorillaWSConn) ReceiveText() (string, error) {
	messageType, data, err := c.conn.ReadMessage()
	if messageType != websocket.TextMessage {
		return "", errors.New("unsupported message type " + strconv.Itoa(messageType))
	}
	return string(data), err
}

func (c *GorillaWSConn) Receive() ([]byte, error) {
	messageType, data, err := c.conn.ReadMessage()
	if messageType != websocket.BinaryMessage {
		return nil, errors.New("unsupported message type " + strconv.Itoa(messageType))
	}
	return data, err
}

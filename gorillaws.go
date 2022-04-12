package cubone

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type GorillaWSConn struct {
	conn *websocket.Conn
	cfg  *GorillaWsConfig
}

func NewGorillaWSConnFactory(cfg *GorillaWsConfig) WSConnFactory {
	u := &websocket.Upgrader{
		HandshakeTimeout:  cfg.HandshakeTimeout,
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		WriteBufferPool:   nil,
		Subprotocols:      nil,
		Error:             nil,
		CheckOrigin:       nil,
		EnableCompression: false,
	}

	return func(w http.ResponseWriter, r *http.Request) (WSConn, error) {
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			return nil, err
		}
		return &GorillaWSConn{conn: conn, cfg: cfg}, nil
	}
}

func (c *GorillaWSConn) Ping() error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *GorillaWSConn) Close() error {
	return c.conn.Close()
}

func (c *GorillaWSConn) Send(data []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.cfg.WriteTimeout))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *GorillaWSConn) Receive() ([]byte, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.WriteTimeout))
	msgType, data, err := c.conn.ReadMessage()
	if msgType != websocket.BinaryMessage {
		return nil, errors.New("unsupported message type " + strconv.Itoa(msgType))
	}
	return data, err
}

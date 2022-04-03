package cubone

import (
	"encoding/json"

	"go.uber.org/atomic"
)

type Client struct {
	ID      string
	wsConn  WSConnection
	readCh  chan<- *WSClientMessage
	writeCh chan []byte
	done    chan struct{}
	closed  *atomic.Bool
}

func NewClient(id string, wsConn WSConnection, readCh chan<- *WSClientMessage) *Client {
	client := &Client{
		ID:      id,
		wsConn:  wsConn,
		readCh:  readCh,
		writeCh: make(chan []byte),
		done:    make(chan struct{}),
		closed:  atomic.NewBool(false),
	}

	return client
}

func (c *Client) startup() error {
	go c.processRead()
	go c.processWrite()

	return nil
}

func (c *Client) write(msg []byte) {
	c.writeCh <- msg
}

func (c *Client) close() error {
	if c.closed.CAS(false, true) {
		c.done <- struct{}{}
		close(c.writeCh)
		return c.wsConn.Close()
	}
	return nil
}

func (c *Client) processWrite() {
	defer c.close()
	for data := range c.writeCh {
		if err := c.wsConn.Send(data); err != nil {
			// Normally this error occurs when handler are trying to send data to a closed connection.
			// TODO: We should disconnect and remove this client ?
			log.Errorw("error while sending message to client", "clientId", c.ID, "error", err.Error())
			break
		}
	}
}

func (c *Client) processRead() {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		default:
			b, err := c.wsConn.Receive()
			if err != nil {
				log.Errorf("reading from ws error: %v", err)
				break
			}

			msg, err := c.parseClientMessage(b)
			if err != nil {
				log.Warnf("received malformed message: %v", err)
				continue
			}

			log.Debugw("received websocket message", "clientId", c.ID, "msg", msg)
			c.readCh <- msg
		}
	}
}

func (c *Client) parseClientMessage(b []byte) (*WSClientMessage, error) {
	var msg WSClientMessage
	err := json.Unmarshal(b, &msg)
	if err != nil {
		return nil, err
	}

	msg.ClientId = c.ID
	return &msg, nil
}

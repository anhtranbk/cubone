package cubone

import (
	"time"

	"go.uber.org/atomic"
)

type clientState int8

type stateChange struct {
	clientId string
	state    clientState
}

// we do not use all of these values here, reversed for the future
// when we need something more complicated such as monitoring
const (
	Created      clientState = 0
	Connected    clientState = 1
	Disconnected clientState = 2
	Error        clientState = 3
)

type Client struct {
	ID      string
	conn    WSConn
	writeCh chan []byte
	readCh  chan<- *WSClientRequest
	stateCh chan<- stateChange
	done    chan struct{}
	closed  *atomic.Bool
}

func NewClient(id string, conn WSConn, rCh chan<- *WSClientRequest, sCh chan<- stateChange) *Client {
	return &Client{
		ID:      id,
		conn:    conn,
		writeCh: make(chan []byte, 128),
		readCh:  rCh,
		stateCh: sCh,
		done:    make(chan struct{}, 1),
		closed:  atomic.NewBool(false),
	}
}

func (c *Client) startup() error {
	go c.processRead()
	go c.processWrite()

	return nil
}

func (c *Client) isClosed() bool {
	return c.closed.Load()
}

func (c *Client) write(payload []byte) {
	c.writeCh <- payload
}

func (c *Client) close() error {
	// make sure client only close once
	if c.closed.CAS(false, true) {
		// notify read goroutine to be stopped
		c.done <- struct{}{}
		// notify write goroutine to be stopped
		close(c.writeCh)
		// notify client manager to remove this client
		c.stateCh <- stateChange{
			clientId: c.ID,
			state:    Disconnected,
		}
		// close underlying websocket connection
		return c.conn.Close()
	}
	return nil
}

// processWrite pumps messages from the writeCh to the websocket connection.
//
// A goroutine running processWrite is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
//
// Besides that, we also send a ping message every second to keep up with alive clients
func (c *Client) processWrite() {
	ticker := time.NewTicker(time.Second)
	defer func() {
		_ = c.close()
		ticker.Stop()
	}()

	for {
		select {
		case data, more := <-c.writeCh:
			if !more {
				// channel was closed
				return
			}
			if err := c.conn.Send(data); err != nil {
				// Normally this error occurs when handler are trying to send data to a closed connection.
				log.Errorw("error while sending message to client", "id", c.ID, "err", err)
				return
			}
		case <-ticker.C:
			if err := c.conn.Ping(); err != nil {
				log.Errorw("ping to client failed", "id", c.ID, "err", err)
				return
			}
		}
	}
}

// processRead read messages from the websocket connection to readCh channel
//
// The application runs processRead in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
//goland:noinspection GoUnhandledErrorResult
func (c *Client) processRead() {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		default:
			b, err := c.conn.Receive()
			if err != nil {
				log.Errorf("reading from ws error: %v", err)
				return
			}

			c.readCh <- &WSClientRequest{ClientId: c.ID, Payload: b}
		}
	}
}

package cubone

import (
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
	ws      WSConnection
	writeCh chan []byte
	readCh  chan<- *WSClientRequest
	stateCh chan<- stateChange
	done    chan struct{}
	closed  *atomic.Bool
}

func NewClient(id string, ws WSConnection, rCh chan<- *WSClientRequest, sCh chan<- stateChange) *Client {
	return &Client{
		ID:      id,
		ws:      ws,
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
		return c.ws.Close()
	}
	return nil
}

//goland:noinspection GoUnhandledErrorResult
func (c *Client) processWrite() {
	defer c.close()
	for data := range c.writeCh {
		if err := c.ws.Send(data); err != nil {
			// Normally this error occurs when handler are trying to send data to a closed connection.
			log.Errorw("error while sending message to client", "clientId", c.ID, "error", err.Error())
			break
		}
	}
}

//goland:noinspection GoUnhandledErrorResult
func (c *Client) processRead() {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		default:
			b, err := c.ws.Receive()
			if err != nil {
				log.Errorf("reading from ws error: %v", err)
				return
			}

			c.readCh <- &WSClientRequest{ClientId: c.ID, Payload: b}
		}
	}
}

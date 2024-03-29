package cubone

import "net/http"

type WSConn interface {
	Ping() error
	Send(data []byte) error
	Receive() ([]byte, error)
	Close() error
}

type WSConnFactory func(w http.ResponseWriter, r *http.Request) (WSConn, error)

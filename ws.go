package cubone

import "net/http"

type WSConnection interface {
	Send(data []byte) error
	Receive() ([]byte, error)
	Close() error
}

type WSConnFactory func(w http.ResponseWriter, r *http.Request) (WSConnection, error)
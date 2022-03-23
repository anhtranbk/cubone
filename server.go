package cubone

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type Server struct {
	server *http.Server
}

func NewServer(config Config) (*Server, error) {
	wsServer, err := NewGorillaWSServer(config.GorillaWS)
	if err != nil {
		return nil, err
	}
	h := &handler{
		wsServer:      wsServer,
		onsiteService: NewOnsiteServiceFromConfig(config),
	}

	router := mux.NewRouter()
	// Url patterns look ugly, but we keep them for compatible with Python version
	router.HandleFunc("/register/{client_id}/{x_client_id}/{x_client_secret}", h.createWebSocket)
	router.HandleFunc("/deregister/{client_id}/{x_client_id}/{x_client_secret}", h.removeWebSocket)
	router.HandleFunc("/internal/onsite/trigger", h.trigger)
	router.HandleFunc("/demo-client", demoClient)

	httpCfg := config.HTTPServer
	server := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%d", httpCfg.Address, httpCfg.Port),
		WriteTimeout: time.Duration(httpCfg.WriteTimeoutInSecond) * time.Second,
		ReadTimeout:  time.Duration(httpCfg.ReadTimeoutInSecond) * time.Second,
	}
	return &Server{server: server}, nil
}

func (s Server) Serve() error {
	log.Infof("Server started listening at %s", s.server.Addr)
	return s.server.ListenAndServe()
}

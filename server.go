package cubone

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	server    *http.Server
	onsiteSvc *OnsiteService
}

func NewServer(cfg Config) (*Server, error) {
	var factory WSConnFactory = nil
	if cfg.GorillaWS != nil {
		factory = NewGorillaWSConnFactory(cfg.GorillaWS)
	}

	if factory == nil {
		log.Fatal("ws config must be provided")
	}
	h := &handler{
		cfg:           cfg,
		wsConnFactory: factory,
		onsiteSvc:     NewOnsiteService(cfg),
	}

	router := mux.NewRouter()
	// Url patterns look ugly, but we keep them for compatible with Python version
	router.HandleFunc("/ws/register/{client_id}/{x_client_id}/{x_client_secret}", h.createWebSocket)
	router.HandleFunc("/ws/deregister/{client_id}/{x_client_id}/{x_client_secret}", h.removeWebSocket)
	router.HandleFunc("/ws/internal/onsite/trigger", h.trigger)
	router.HandleFunc("/demo-client", h.demoClient)

	httpCfg := cfg.HTTPServer
	server := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%d", httpCfg.Address, httpCfg.Port),
		WriteTimeout: time.Duration(httpCfg.WriteTimeout) * time.Second,
		ReadTimeout:  time.Duration(httpCfg.ReadTimeout) * time.Second,
	}
	return &Server{server: server, onsiteSvc: h.onsiteSvc}, nil
}

func (s *Server) Serve() error {
	s.onsiteSvc.Start()
	log.Infof("http server started listening at %s", s.server.Addr)
	return s.server.ListenAndServe()
}

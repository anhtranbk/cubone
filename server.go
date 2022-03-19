package cubone

import (
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type WSServer interface {
	NewConnection(w http.ResponseWriter, r *http.Request) (WebSocketConnection, error)
}

type handler struct {
	wsServer      WSServer
	onsiteService *onsiteService
}

type Server struct {
	server *http.Server
}

func NewServer(config Config) (*Server, error) {
	wsServer, err := NewGorillaWSServer(config.GorillaWS)
	if err != nil {
		return nil, err
	}
	handler := &handler{
		wsServer:      wsServer,
		onsiteService: NewOnsiteServiceFromConfig(config),
	}

	r := mux.NewRouter()
	// Url patterns look ugly, but we keep them for compatible with Python version
	r.HandleFunc("/register/{client_id}/{x_client_id}/{x_client_secret}", handler.createWebSocket)
	r.HandleFunc("/deregister/{client_id}/{x_client_id}/{x_client_secret}", handler.removeWebSocket)
	r.HandleFunc("/internal/onsite/trigger", handler.trigger)

	httpConfig := config.HTTPServer
	server := &http.Server{
		Handler: r,
		Addr:    httpConfig.Address,
		WriteTimeout: time.Duration(httpConfig.WriteTimeoutInSecond) * time.Second,
		ReadTimeout:  time.Duration(httpConfig.ReadTimeoutInSecond) * time.Second,
	}
	return &Server{server: server}, nil
}

func (s Server) Serve() error {
	log.Infof("Server started listening at %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// /register/{client_id}/{x_client_id}/{x_client_access_token}
func (h handler) createWebSocket(w http.ResponseWriter, r *http.Request) {
	panic("implement me")
}

// /deregister/{client_id}/{x_client_id}/{x_client_access_token}
func (h handler) removeWebSocket(w http.ResponseWriter, r *http.Request) {
	panic("implement me")
}

// /internal/onsite/trigger
func (h handler) trigger(w http.ResponseWriter, r *http.Request) {
	panic("implement me")
}

func handleRequestAuthentication(w http.ResponseWriter, r *http.Request) bool {
	params := mux.Vars(r)
	xClientId := params["x_client_id"]
	xClientSecret := params["x_client_secret"]
	ok, err := authenticate(xClientId, xClientSecret)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "invalid credentials")
		return false
	}
	if !ok {
		w.WriteHeader(401)
		_, _ = w.Write([]byte("invalid credentials"))
		return false
	}
	return true
}

func authenticate(clientId string, clientSecret string) (bool, error) {
	// TODO implement authentication here
	return true, nil
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMessage string) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(errorMessage))
}

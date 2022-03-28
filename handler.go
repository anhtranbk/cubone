package cubone

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

const htmlTemplate = `
<!DOCTYPE html>
<html>
    <head>
        <title>Chat</title>
    </head>
    <body>
        <h1>WebSocket Chat</h1>
        <h2>Your ID: <span id="ws-id"></span></h2>
        <form action="" onsubmit="sendMessage(event)">
            <input type="text" id="messageText" autocomplete="off"/>
            <button>Send</button>
        </form>
        <ul id='messages'>
        </ul>
        <script>
            let params = (new URL(document.location)).searchParams;
            var clientId = params.get("clientId");
            console.log(clientId);
            if (clientId === null || clientId === "") {
                clientId = Date.now()
            }
            document.querySelector("#ws-id").textContent = clientId;
            var ws = new WebSocket("ws://localhost:%d/ws/register/" + clientId + "/abc/xyz");

            const messageOrders = new Map()
            const maxMessagePerOrder = 8

            ws.onmessage = function(event) {
                var messages = document.getElementById('messages')
                var message = document.createElement('li')
                var content = document.createTextNode(event.data)
                message.appendChild(content)
                messages.appendChild(message)

                var words = event.data.split(' ')
                var last = words[words.length-1]

                if (messageOrders.get(last) != undefined) {
                    messageOrders.set(last, messageOrders.get(last) + 1)
                    const length = messageOrders.get(last)
                    if (length > maxMessagePerOrder) {
                        console.error('Duplicate message at order: ' + last)
                    } else if (length == maxMessagePerOrder) {
                        console.log('Order ' + last + ' has reached maximum value')
                    }
                } else {
                    messageOrders.set(last, 1)
                }
            };
            function sendMessage(event) {
                var input = document.getElementById("messageText")
                ws.send(input.value)
                input.value = ''
                event.preventDefault()
            }
        </script>
    </body>
</html>
`

type WSServer interface {
	NewConnection(w http.ResponseWriter, r *http.Request) (WebSocketConnection, error)
}

type handler struct {
	cfg       Config
	wsServer  WSServer
	onsiteSvc *OnsiteService
}

// /register/{client_id}/{x_client_id}/{x_client_access_token}
func (h *handler) createWebSocket(w http.ResponseWriter, r *http.Request) {
	if ok := handleAuthentication(w, r); !ok {
		return
	}

	conn, err := h.wsServer.NewConnection(w, r)
	if err != nil {
		log.Errorw("could not open websocket connection", "err", err.Error())
		writeErrorResponse(w, http.StatusInternalServerError, "could not create websocket connection")
		return
	}

	params := mux.Vars(r)
	clientId := params["client_id"]
	if err := h.onsiteSvc.ConnectClient(clientId, conn); err != nil {
		log.Errorw("could not open websocket connection", "error", err.Error())
		writeErrorResponse(w, http.StatusInternalServerError, "could not create websocket connection")
	}
}

// /deregister/{client_id}/{x_client_id}/{x_client_access_token}
func (h *handler) removeWebSocket(w http.ResponseWriter, r *http.Request) {
	if ok := handleAuthentication(w, r); !ok {
		return
	}

	params := mux.Vars(r)
	clientId := params["client_id"]
	if err := h.onsiteSvc.DisconnectClient(clientId); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "could not remove websocket connection")
		log.Errorw("could not close websocket connection", "error", err.Error())
	}
}

// /internal/onsite/trigger
func (h *handler) trigger(w http.ResponseWriter, r *http.Request) {
	reader, err := r.GetBody()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "could not read trigger request")
		return
	}
	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "could not read trigger request")
		return
	}

	var msg DeliveryMessage
	if err := json.Unmarshal(buffer, &msg); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "could not parse trigger request")
		return
	}

	if err := h.onsiteSvc.PublishMessage(&msg); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "could not publish message")
		return
	}
}

func (h *handler) demoClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	_, _ = w.Write([]byte(fmt.Sprintf(htmlTemplate, h.cfg.HTTPServer.Port)))
}

func handleAuthentication(w http.ResponseWriter, r *http.Request) bool {
	params := mux.Vars(r)
	id := params["x_client_id"]
	secret := params["x_client_secret"]
	ok, err := authenticate(id, secret)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "invalid credentials")
		return false
	}
	return true
}

func authenticate(clientId string, clientSecret string) (bool, error) {
	// TODO implement authentication here or forward to a remote auth service
	return clientId != "" && clientSecret != "", nil
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(msg))
}

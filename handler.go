package cubone

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

const html = `
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
            var ws = new WebSocket("ws://localhost:11053/prile/ws/register/${clientId}/abc/xyz");

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
	wsServer      WSServer
	onsiteService *onsiteService
}

// /register/{client_id}/{x_client_id}/{x_client_access_token}
func (h handler) createWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.wsServer.NewConnection(w, r)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not create websocket connection")
		log.Errorw("Could not create websocket connection", "error", err.Error())
		return
	}

	if ok := handleRequestAuthentication(w, r); !ok {
		return
	}

	params := mux.Vars(r)
	clientId := params["client_id"]
	err = h.onsiteService.ConnectClient(clientId, conn)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not create websocket connection")
		log.Errorw("Could not create websocket connection", "error", err.Error())
	}
}

// /deregister/{client_id}/{x_client_id}/{x_client_access_token}
func (h handler) removeWebSocket(w http.ResponseWriter, r *http.Request) {
	if ok := handleRequestAuthentication(w, r); !ok {
		return
	}

	params := mux.Vars(r)
	clientId := params["client_id"]
	err := h.onsiteService.DisconnectClient(clientId)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not remove websocket connection")
		log.Errorw("Could not remove websocket connection", "error", err.Error())

	}
}

// /internal/onsite/trigger
func (h handler) trigger(w http.ResponseWriter, r *http.Request) {
	reader, err := r.GetBody()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Could not read body request")
		return
	}
	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Could not read body request")
		return
	}

	var message DeliveryMessage
	err = json.Unmarshal(buffer, &message)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Could not parse body request")
		return
	}

	err = h.onsiteService.PublishMessage(&message)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "handler could not publish message")
		log.Errorw("Could not publish message", "error", err.Error())
		return
	}
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
	// TODO implement authentication here or add an authenticate service
	return true, nil
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMessage string) {
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(errorMessage))
}

func demoClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	_, _ = w.Write([]byte(html))
}

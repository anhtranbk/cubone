package cubone

// WebSocket client request message
type WebSocketClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WebSocket handler response message
type WebSocketServerMessage struct {
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

type OnsiteMessage struct {
	Endpoint string
	Data     interface{}
}

type MembershipMessage struct {
	ClientId string
	OwnerId  string
}



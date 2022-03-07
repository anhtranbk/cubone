package cubone

// WSClientMessage client request message
type WSClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WSServerMessage handler response message
type WSServerMessage struct {
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

type DeliveryMessage struct {
	Endpoint string
	Data     interface{}
}

type MembershipMessage struct {
	ClientId string
	OwnerId  string
}



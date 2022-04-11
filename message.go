package cubone

type WSClientRequest struct {
	ClientId string
	Payload  []byte
}

// WSClientMessage client request message
type WSClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Ack  *AckMessage
}

// WSServerMessage handler response message
type WSServerMessage struct {
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

type DeliveryMessage struct {
	Endpoint string      `json:"endpoint"`
	Data     interface{} `json:"data"`
}

type MembershipMessage struct {
	ClientId string `json:"client_id"`
	OwnerId  string `json:"owner_id"`
}

type AckMessage struct {
	ID string `json:"id"`
}

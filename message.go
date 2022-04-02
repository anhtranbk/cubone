package cubone

// WSClientMessage client request message
type WSClientMessage struct {
	ClientId string
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
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

type AckMessage struct {
	ID string
}

func NewDeliveryMessage(msg *PubSubMessage) *DeliveryMessage {
	// pub-sub messages are only sent by other servers, so we don't need to care about invalid message format here
	data := msg.Data.(map[string]interface{})
	return &DeliveryMessage{
		Endpoint: data["endpoint"].(string),
		Data:     data["data"],
	}
}

func NewMembershipMessage(msg *PubSubMessage) *MembershipMessage {
	// pub-sub messages are only sent by other servers, so we don't need to care about invalid message format here
	data := msg.Data.(map[string]interface{})
	return &MembershipMessage{
		ClientId: data["client_id"].(string),
		OwnerId:  data["owner_id"].(string),
	}
}

func NewAckMessage(msg *WSClientMessage) (*AckMessage, error) {
	// TODO: add error handling
	data, _ := msg.Data.(map[string]interface{})
	return &AckMessage{
		ID: data["id"].(string),
	}, nil
}

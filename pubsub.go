package cubone

type PubSubMessage struct {
	Type    string
	Channel string
	Data    interface{}
}

type Publisher interface {
	Publish(channel string, data interface{}) error
}

type Subscriber interface {
	Subscribe(channel string) error
	GetMessage(timeout int64) (*PubSubMessage, error)
}

type FakePubSub struct {
}

func (f FakePubSub) Subscribe(channel string) error {
	return nil
}

func (f FakePubSub) GetMessage(timeout int64) (*PubSubMessage, error) {
	return &PubSubMessage{
		Type:    "message",
		Channel: OnsiteChannel,
		Data: struct {
			name    string
			age     int
			address string
		}{
			name:    "nhat anh",
			age:     33,
			address: "q7, hcm",
		},
	}, nil
}

func (f FakePubSub) Publish(channel string, data interface{}) error {
	log.Infow("message published", "channel", channel, "data", data)
	return nil
}

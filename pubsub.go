package cubone

type PubSubMessage struct {
	Channel string
	Payload []byte
}

type PubSub interface {
	Publish(channel string, data []byte) error
	Subscribe(channels ...string) error
	Channel() <-chan *PubSubMessage
	Close() error
}

type FakePubSub struct {
	ch chan *PubSubMessage
}

func NewFakePubSub() *FakePubSub {
	fps := &FakePubSub{
		ch: make(chan *PubSubMessage),
	}
	return fps
}

func (f *FakePubSub) Subscribe(channels ...string) error {
	return nil
}

func (f *FakePubSub) Channel() <-chan *PubSubMessage {
	return f.ch
}

func (f *FakePubSub) Publish(channel string, data []byte) error {
	log.Debugw("message published", "channel", channel, "data", string(data))
	msg := &PubSubMessage{
		Channel: channel,
		Payload: data,
	}
	f.ch <- msg
	return nil
}

func (f *FakePubSub) Close() error {
	log.Info("fake pubsub closed")
	return nil
}

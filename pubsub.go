package cubone

type PubSubMessage struct {
	Channel string
	Data    interface{}
}

type Publisher interface {
	Publish(channel string, data interface{}) error
	Close() error
}

type Subscriber interface {
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

func (f *FakePubSub) Publish(channel string, data interface{}) error {
	log.Debugw("message published", "channel", channel, "data", data)
	msg := &PubSubMessage{
		Channel: channel,
		Data:    data,
	}
	f.ch <- msg
	return nil
}

func (f *FakePubSub) Close() error {
	log.Info("fake pubsub closed")
	return nil
}

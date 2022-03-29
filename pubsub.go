package cubone

import "time"

type PubSubMessage struct {
	Channel string
	Data    interface{}
}

type Publisher interface {
	Publish(channel string, data interface{}) error
}

type Subscriber interface {
	Subscribe(channels ...string) error
	Channel() <-chan *PubSubMessage
}

type FakePubSub struct {
	ch chan *PubSubMessage
}

func NewFakePubSub() *FakePubSub {
	fps := &FakePubSub{
		ch: make(chan *PubSubMessage),
	}
	go fps.genFakeMessages()
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
	return nil
}

func (f *FakePubSub) genFakeMessages() {
	for {
		msg := &PubSubMessage{
			Channel: OnsiteChannel,
			Data: struct {
				name string
				age  int
				addr string
			}{
				name: "nhat anh",
				age:  32,
				addr: "q7, hcm",
			},
		}
		f.ch <- msg
		time.Sleep(time.Second)
	}
}

package cubone

import (
	"errors"
	"strings"

	"github.com/go-redis/redis"
)

type RedisPubSub struct {
	client redis.UniversalClient
	ch     chan *PubSubMessage
}

type RedisConfig struct {
	RedisOptions        *redis.Options
	RedisClusterOptions *redis.ClusterOptions
}

func NewRedisPubSubFromConnStr(connStr string) (*RedisPubSub, error) {
	parts := strings.Split(connStr, ",")
	var client redis.UniversalClient
	if len(parts) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{Addrs: parts})
	} else {
		client = redis.NewClient(&redis.Options{Addr: parts[0]})
	}
	return &RedisPubSub{
		client: client,
		ch:     nil,
	}, nil
}

func NewRedisPubSub(cfg *RedisConfig) (*RedisPubSub, error) {
	r := &RedisPubSub{client: nil}
	if cfg.RedisOptions != nil {
		r.client = redis.NewClient(cfg.RedisOptions)
	} else if cfg.RedisClusterOptions != nil {
		r.client = redis.NewClusterClient(cfg.RedisClusterOptions)
	} else {
		return nil, errors.New("at least one redis options must be provided")
	}
	return r, nil
}

func (r *RedisPubSub) Subscribe(channels ...string) error {
	pubsub := r.client.Subscribe(channels...)
	go func() {
		defer func(ps *redis.PubSub) {
			_ = ps.Close()
		}(pubsub)

		for msg := range pubsub.Channel() {
			pubsubMsg := &PubSubMessage{
				Channel: msg.Channel,
				Data:    msg.Payload,
			}
			r.ch <- pubsubMsg
		}
	}()
	return nil
}

func (r *RedisPubSub) Channel() <-chan *PubSubMessage {
	return r.ch
}

func (r *RedisPubSub) Publish(channel string, data interface{}) error {
	_, err := r.client.Publish(channel, data).Result()
	return err
}

func (r *RedisPubSub) Close() error {
	if err := r.client.Close(); err != nil {
		return err
	}
	return nil
}

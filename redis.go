package cubone

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
)

type RedisPubSub struct {
	client redis.UniversalClient
	ch     chan *PubSubMessage
}

func NewRedisPubSubFromAddrStr(addrStr string) (*RedisPubSub, error) {
	addrs := strings.Split(addrStr, ",")
	var client redis.UniversalClient
	if len(addrs) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{Addrs: addrs})
	} else {
		client = redis.NewClient(&redis.Options{Addr: addrs[0]})
	}

	if err := verifyClient(client); err != nil {
		return nil, fmt.Errorf("failed to verify client: %v", err)
	}

	log.Infof("connected to redis at: %v", addrs)
	return &RedisPubSub{
		client: client,
		ch:     make(chan *PubSubMessage, 1024),
	}, nil
}

func NewRedisPubSub(cfg *RedisConfig) (*RedisPubSub, error) {
	var client redis.UniversalClient
	if cfg.RedisOptions != nil {
		client = redis.NewClient(cfg.RedisOptions)
	} else if cfg.RedisClusterOptions != nil {
		client = redis.NewClusterClient(cfg.RedisClusterOptions)
	} else {
		return nil, errors.New("at least one redis options must be provided")
	}

	if err := verifyClient(client); err != nil {
		return nil, fmt.Errorf("failed to verify client: %v", err)
	}

	if cfg.RedisClusterOptions != nil {
		log.Info("connected to single redis instance at: %s", cfg.RedisClusterOptions.Addrs)
	} else {
		log.Info("connected to redis cluster at: %s", cfg.RedisClusterOptions.Addrs)
	}

	return &RedisPubSub{
		client: client,
		ch:     make(chan *PubSubMessage, 1024),
	}, nil
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
				Payload: []byte(msg.Payload),
			}
			r.ch <- pubsubMsg
		}
	}()
	return nil
}

func (r *RedisPubSub) Channel() <-chan *PubSubMessage {
	return r.ch
}

func (r *RedisPubSub) Publish(channel string, data []byte) error {
	_, err := r.client.Publish(channel, data).Result()
	return err
}

func (r *RedisPubSub) Close() error {
	if err := r.client.Close(); err != nil {
		return err
	}
	return nil
}

func verifyClient(client redis.UniversalClient) error {
	return client.Ping().Err()
}

package cubone

import (
	"time"

	"github.com/go-redis/redis"
)

type Config struct {
	// app configuration
	MessageRetryTimeout  time.Duration
	MessageRetryInterval time.Duration
	ConnectionTimeout    time.Duration
	RedisAddr            string

	// module specified configurations
	GorillaWS  *GorillaWsConfig
	HTTPServer *HTTPServerConfig
	Redis      *RedisConfig
}

type GorillaWsConfig struct {
	ReadBufferSize   int
	WriteBufferSize  int
	HandshakeTimeout time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
}

type HTTPServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type RedisConfig struct {
	RedisOptions        *redis.Options
	RedisClusterOptions *redis.ClusterOptions
}

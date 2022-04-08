package cubone

import (
	"time"

	"github.com/go-redis/redis"
)

type Config struct {
	// app configuration
	MessageRetryMaxTimeout time.Duration
	MessageRetryDelay      time.Duration
	ConnectionTimeout      time.Duration
	RedisAddr              string

	// module specified configurations
	GorillaWS  *GorillaWsConfig
	HTTPServer *HTTPServerConfig
	Redis      *RedisConfig
}

type GorillaWsConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type HTTPServerConfig struct {
	Address      string
	Port         uint16
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type RedisConfig struct {
	RedisOptions        *redis.Options
	RedisClusterOptions *redis.ClusterOptions
}

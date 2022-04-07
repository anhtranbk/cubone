package cubone

import "time"

type Config struct {
	MessageRetryMaxTimeout time.Duration
	MessageRetryDelay      time.Duration
	ConnectionTimeout      time.Duration
	GorillaWS              *GorillaWsConfig
	HTTPServer             *HTTPServerConfig
}

type GorillaWsConfig struct {
	ReadBufferSize  uint32
	WriteBufferSize uint32
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type HTTPServerConfig struct {
	Address      string
	Port         uint16
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

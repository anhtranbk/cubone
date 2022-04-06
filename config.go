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
	ReadBufferSize  int
	WriteBufferSize int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type HTTPServerConfig struct {
	Address              string
	Port                 int
	ReadTimeoutInSecond  int
	WriteTimeoutInSecond int
}

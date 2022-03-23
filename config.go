package cubone

type Config struct {
	MessageRetryMaxTimeout int64
	MessageRetryDelay      int64
	GorillaWS              GorillaWsConfig
	HTTPServer             HTTPServerConfig
}

type GorillaWsConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
}

type HTTPServerConfig struct {
	Address              string
	Port                 int
	ReadTimeoutInSecond  int
	WriteTimeoutInSecond int
}

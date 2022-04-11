package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/anhtranbk/cubone"
)

func getPortFromEnvOrDefault(defaultPort uint16) uint16 {
	v := os.Getenv("PORT")
	if v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid PORT env: %v\n", v)
		}
		if port < 1025 || port > 65535 {
			log.Fatalf("port value must be between 1025-65535")
		}
		return uint16(port)
	}
	return defaultPort
}

func main() {
	server, err := cubone.NewServer(cubone.Config{
		MessageRetryTimeout:  time.Second * 60,
		MessageRetryInterval: time.Second * 2,
		ConnectionTimeout:    time.Second * 5,
		RedisAddr:            os.Getenv("REDIS_ADDRESS"),
		GorillaWS: &cubone.GorillaWsConfig{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		},
		HTTPServer: &cubone.HTTPServerConfig{
			Address:      "0.0.0.0",
			Port:         getPortFromEnvOrDefault(11888),
			ReadTimeout:  time.Second * 5,
			WriteTimeout: time.Second * 5,
		},
	})
	if err != nil {
		log.Fatalln("server error", err.Error())
	}

	log.Fatal(server.Serve())
}

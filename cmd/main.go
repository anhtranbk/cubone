package main

import (
	"log"
	"os"
	"time"

	"github.com/anhtranbk/cubone"
)

func main() {
	server, err := cubone.NewServer(cubone.Config{
		MessageRetryTimeout:  time.Second * 60,
		MessageRetryInterval: time.Second * 2,
		ConnectionTimeout:    time.Second * 5,
		RedisAddr:            os.Getenv("REDIS_ADDRESS"),
		GorillaWS: &cubone.GorillaWsConfig{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
			ReadTimeout:     time.Second * 10,
			WriteTimeout:    time.Second * 10,
		},
		HTTPServer: &cubone.HTTPServerConfig{
			Addr:         os.Getenv("ADDR"),
			ReadTimeout:  time.Second * 5,
			WriteTimeout: time.Second * 5,
		},
	})
	if err != nil {
		log.Fatalln("server error", err.Error())
	}

	log.Fatal(server.Serve())
}

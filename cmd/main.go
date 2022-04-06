package main

import (
	"log"
	"time"

	"github.com/anhtranbk/cubone"
)

func main() {
	server, err := cubone.NewServer(cubone.Config{
		MessageRetryMaxTimeout: time.Second * 120,
		MessageRetryDelay:      time.Second * 5,
		ConnectionTimeout:      time.Second * 10,
		GorillaWS: &cubone.GorillaWsConfig{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		},
		HTTPServer: &cubone.HTTPServerConfig{
			Address:              "localhost",
			Port:                 11053,
			ReadTimeoutInSecond:  5,
			WriteTimeoutInSecond: 5,
		},
	})
	if err != nil {
		log.Fatalln("server error", err.Error())
	}

	log.Fatal(server.Serve())
}

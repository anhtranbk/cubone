package main

import (
	"github.com/anhtranbk/cubone"
	"log"
)

func main() {
	server, err := cubone.NewServer(cubone.Config{
		MessageRetryMaxTimeout: 5,
		MessageRetryDelay:      5,
		GorillaWS: cubone.GorillaWsConfig{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		},
		HTTPServer: cubone.HTTPServerConfig{
			Address:              "localhost",
			Port: 11053,
			ReadTimeoutInSecond:  5,
			WriteTimeoutInSecond: 5,
		},
	})
	if err != nil {
		log.Fatalln("Server error", err.Error())
	}

	log.Fatal(server.Serve())
}

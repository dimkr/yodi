package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/dimkr/yodi/pkg/mqtt"
)

const (
	defaultPort = "2883"
)

func handle(broker *mqtt.Broker, conn net.Conn) {
	defer conn.Close()

	client, err := broker.NewClient(conn)
	if err != nil {
		return
	}
	defer client.Close()

	client.Run()
}

func main() {
	log.SetLevel(log.WarnLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	broker, err := mqtt.NewBroker()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go handle(broker, conn)
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

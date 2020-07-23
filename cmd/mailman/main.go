package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/dimkr/yodi/pkg/mqtt"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		redisClient, err := mqtt.ConnectToRedis(ctx)
		if err != nil {
			log.Fatal(err)
		}

		store := mqtt.NewStore(redisClient)

		for {
			queuedMessage, err := store.PopQueuedMessage(ctx)
			if err != nil {
				log.Fatal(err)
			}

			err = store.QueueMessageForSubscribers(queuedMessage)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

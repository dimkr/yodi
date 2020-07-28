// This file is part of yodi.
//
// Copyright 2020 Dima Krasner
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	log.SetLevel(log.WarnLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		redisClient, err := mqtt.ConnectToRedis(ctx)
		if err != nil {
			log.Fatal(err)
		}

		store, err := mqtt.NewRedisStore(ctx, redisClient)
		if err != nil {
			log.Fatal(err)
		}

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

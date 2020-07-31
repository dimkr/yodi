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
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/dimkr/yodi/pkg/mqtt"
	"github.com/dimkr/yodi/pkg/store"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := store.NewRedisStore(ctx)
	if err != nil {
		log.Fatal(err)
	}

	broker, err := mqtt.NewBroker(ctx, store)
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

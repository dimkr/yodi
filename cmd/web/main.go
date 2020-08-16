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
	"net/http"
	"os"

	"github.com/dimkr/yodi/pkg/mqtt"
	"github.com/dimkr/yodi/pkg/store"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	upgrader = websocket.Upgrader{Subprotocols: []string{"mqtt"}}
	broker   *mqtt.Broker
)

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleMQTT(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client, err := mqtt.NewWebSocketClient(r.Context(), conn, broker)
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
		port = "8080"
	}

	http.HandleFunc("/health", handleHealthCheck)
	http.HandleFunc("/mqtt", handleMQTT)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/static"))))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := store.NewRedisStore(ctx)
	if err != nil {
		log.Fatal(err)
	}

	broker, err = mqtt.NewBroker(ctx, store)
	if err != nil {
		log.Fatal(err)
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

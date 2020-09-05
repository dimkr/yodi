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
	"strings"

	"github.com/dimkr/yodi/pkg/mqtt"
	"github.com/dimkr/yodi/pkg/store"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

var (
	upgrader = websocket.Upgrader{Subprotocols: []string{"mqtt"}}
	broker   *mqtt.Broker
)

func handleHealthCheck(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func handleMQTT(c echo.Context) error {
	r := c.Request()

	conn, err := upgrader.Upgrade(c.Response().Writer, r, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := mqtt.NewWebSocketClient(r.Context(), conn, broker)
	if err != nil {
		return err
	}
	defer client.Close()

	client.Run()
	return nil
}

func main() {
	log.SetLevel(log.WarnLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e := echo.New()
	if e == nil {
		log.Fatal("e is nil")
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", handleHealthCheck)
	e.GET("/mqtt", handleMQTT)
	e.Static("/static", "/static")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := store.NewRedisStore(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	auth := mqtt.NewAuthenticator(store)

	authMiddleware := middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Validator: func(username, password string, c echo.Context) (bool, error) {
			_, err := auth.AuthenticateUser(c.Request().Context(), username, password)
			if err != nil {
				return false, nil
			}
			return true, nil
		},
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Request().URL.Path, "/static")
		},
	})
	e.Use(authMiddleware)

	broker, err = mqtt.NewBroker(ctx, store, auth)
	if err != nil {
		log.Fatal(err)
	}

	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

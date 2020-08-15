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

package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	clientID             string
	logFields            log.Fields
	reader               io.Reader
	writer               io.Writer
	ctx                  context.Context
	cancel               context.CancelFunc
	startMessagesRoutine sync.Once
	registered           bool
	broker               *Broker
	messageQueue         chan *QueuedMessage
	lastPingTime         time.Time
}

const (
	connectionTimeout = time.Hour
	ackTimeout        = time.Second * 5

	maxTopicLength    = 64
	maxUsernameLength = 64
	maxPasswordLength = 64
)

var ErrDisconnected = errors.New("Client has disconnected")

func NewWebSocketClient(parent context.Context, conn *websocket.Conn, broker *Broker) (*Client, error) {
	return NewClient(parent, wrapWebSocket(conn), broker)
}

func NewClient(parent context.Context, conn net.Conn, broker *Broker) (*Client, error) {
	t := time.Now().Add(connectionTimeout)
	if err := conn.SetDeadline(t); err != nil {
		return nil, err
	}

	messageQueue := make(chan *QueuedMessage, 1)

	ctx, cancel := context.WithDeadline(parent, t)
	return &Client{
		logFields:    log.Fields{},
		reader:       conn,
		writer:       conn,
		ctx:          ctx,
		cancel:       cancel,
		broker:       broker,
		messageQueue: messageQueue,
	}, nil
}

func (c *Client) Close() {
	if c.registered {
		if err := c.broker.RemoveClient(c.clientID); err != nil {
			log.WithError(err).Warn("Failed to remove a client")
		}
	}

	c.cancel()
}

func (c *Client) deliverMessages() {
	log.WithFields(c.logFields).Info("Starting message delivery routine")

	for {
		select {
		case <-c.ctx.Done():
			log.WithFields(c.logFields).Info("Stopping message delivery routine")
			return

		case queuedMessage := <-c.messageQueue:
			// we don't want to requeue the message while we're sending it
			if queuedMessage.QoS != QoS0 {
				queuedMessage.SendTime = time.Now()
				queuedMessage.Duplicate = true
				if err := c.broker.UpdateQueuedMessageForSubscriber(c.ctx, c.clientID, queuedMessage); err != nil {
					continue
				}
			}

			c.publish(queuedMessage)
		}
	}
}

func (c *Client) queueMessages() {
	messagesChannel := c.broker.GetMessagesChannelForSubscriber(c.ctx, c.clientID)

	for {
		select {
		case <-c.ctx.Done():
			return

		case queuedMessage := <-messagesChannel:
			c.messageQueue <- queuedMessage

		case <-time.After(ackTimeout):
			now := time.Now()

			c.broker.ScanQueuedMessagesForSubscriber(c.ctx, c.clientID, func(queuedMessage *QueuedMessage) {
				if !queuedMessage.Duplicate || now.After(queuedMessage.SendTime.Add(ackTimeout)) {
					c.messageQueue <- queuedMessage
				}
			})
		}
	}
}

func (c *Client) readPacket() error {
	flags := make([]byte, 1)
	if _, err := c.reader.Read(flags); err != nil {
		return err
	}

	length, err := c.readRemainingLength()
	if err != nil {
		return err
	}

	hdr := Header{Flags: flags[0], MessageLength: length}
	messageType := MessageType(hdr.Flags >> 4)

	if !c.registered {
		if messageType != Connect {
			return fmt.Errorf("must connect first")
		}

		return c.readConnect(hdr)
	}

	switch messageType {
	case Publish:
		return c.readPublish(hdr)

	case PublishAck:
		return c.readPublishAck()

	case PingRequest:
		return c.readPing(hdr)

	case PingResponse:
		return c.readPingResponse(hdr)

	case Subscribe:
		return c.readSubscribe()

	case Unsubscribe:
		return c.readUnsubscribe()

	case Disconnect:
		return ErrDisconnected

	default:
		return fmt.Errorf("unknown message type %d", messageType)
	}
}

func (c *Client) Run() error {
	for {
		if err := c.readPacket(); err != nil {
			if !errors.Is(err, ErrDisconnected) {
				log.WithFields(c.logFields).Warn(err)
			}
			return err
		}
	}
}

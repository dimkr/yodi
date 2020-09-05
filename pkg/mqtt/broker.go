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
	"encoding/json"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/dimkr/yodi/pkg/store"
	log "github.com/sirupsen/logrus"
)

// Broker is an MQTT broker
type Broker struct {
	store store.Store
	ctx   context.Context
	auth  Authenticator
}

const (
	clientSet                    = "/clients"
	messageQueue                 = "/messages"
	topicSubscribersSetFmt       = "/topic/%s/subscribers"
	clientSubscriptionsSetFmt    = "/client/%s/subscriptions"
	clientMessageQueueFmt        = "/client/%s/messages"
	clientMessageNotificationFmt = "/client/%s/notify"
)

// NewBroker creates a new MQTT broker
func NewBroker(ctx context.Context, store store.Store, auth Authenticator) (*Broker, error) {
	return &Broker{store: store, ctx: ctx, auth: auth}, nil
}

// NewClient creates a new MQTT client connected to a broker
func (b *Broker) NewClient(conn net.Conn) (*Client, error) {
	return NewClient(b.ctx, conn, b)
}

// AddClient registers an authenticated MQTT client
func (b *Broker) AddClient(ctx context.Context, clientID string) error {
	return b.store.Set(clientSet).Add(ctx, clientID)
}

// RemoveClient unregisters an authenticated MQTT client
func (b *Broker) RemoveClient(clientID string) error {
	topics, err := b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Members(b.ctx)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, topic)).Remove(b.ctx, clientID)
	}

	if err := b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Destroy(b.ctx); err != nil {
		return err
	}

	if err := b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Destroy(b.ctx); err != nil {
		return err
	}

	if err := b.store.Queue(fmt.Sprintf(clientMessageNotificationFmt, clientID)).Destroy(b.ctx); err != nil {
		return err
	}

	if err := b.store.Set(clientSet).Remove(b.ctx, clientID); err != nil {
		return err
	}

	return nil
}

// Subscribe subscribes an MQTT client to a topic
func (b *Broker) Subscribe(ctx context.Context, clientID, topic string) error {
	if err := b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Add(ctx, topic); err != nil {
		return err
	}

	err := b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, topic)).Add(ctx, clientID)
	if err != nil {
		b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Remove(ctx, topic)
	}

	return err
}

// Unsubscribe unsubscribes an MQTT client from a topic
func (b *Broker) Unsubscribe(ctx context.Context, clientID, topic string) error {
	if err := b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, topic)).Remove(ctx, clientID); err != nil {
		return err
	}

	return b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Remove(ctx, topic)
}

func encodeMessage(queuedMessage *QueuedMessage) (string, error) {
	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return "", err
	}

	return string(j), nil
}

func decodeMessage(raw string) (*QueuedMessage, error) {
	var msg QueuedMessage
	err := json.Unmarshal([]byte(raw), &msg)
	if err != nil {
		log.WithFields(log.Fields{"raw": raw}).WithError(err).Warn("failed to decode a message")
		return nil, err
	}

	return &msg, nil
}

func (b *Broker) popMessage(ctx context.Context, queue string) (*QueuedMessage, error) {
	result, err := b.store.Queue(queue).Pop(ctx)
	if err != nil {
		return nil, err
	}

	return decodeMessage(result)
}

// PopQueuedMessage pops one message from the queue of published messages
func (b *Broker) PopQueuedMessage(ctx context.Context) (*QueuedMessage, error) {
	return b.popMessage(ctx, messageQueue)
}

// QueueMessage pushes a message into the queue of published messages
func (b *Broker) QueueMessage(topic string, msg string, messageID uint16, qos QoS) error {
	queuedMessage := QueuedMessage{ID: messageID, Topic: topic, Message: msg, QoS: qos}

	log.WithFields(queuedMessage.LogFields()).Info("Queueing a message")

	j, err := encodeMessage(&queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	err = b.store.Queue(messageQueue).Push(b.ctx, j)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to queue a message")
	}

	return err
}

// QueueMessageForSubscribers pushes a published message into the message queue
// of each client subscribed to the topic the message was published to
func (b *Broker) QueueMessageForSubscribers(queuedMessage *QueuedMessage) error {
	return b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, queuedMessage.Topic)).Scan(b.ctx, func(ctx context.Context, clientID string) {
		b.QueueMessageForSubscriber(ctx, clientID, queuedMessage)
	})
}

// UnqueueMessageForSubscriber removes a published message from the messages
// queue of a client
func (b *Broker) UnqueueMessageForSubscriber(ctx context.Context, clientID string, messageID uint16) error {
	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Remove(ctx, fmt.Sprintf("%d", messageID))
}

func generateMessageID() uint16 {
	// TODO: is this unique enough?
	return uint16(time.Now().UnixNano() % math.MaxUint16)
}

// QueueMessageForSubscriber pushes a published message into the message queue
// of a client
func (b *Broker) QueueMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	queuedMessageForSubscriber := *queuedMessage
	queuedMessageForSubscriber.ID = generateMessageID()

	j, err := encodeMessage(&queuedMessageForSubscriber)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	if queuedMessage.QoS != QoS0 {
		err := b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Set(ctx, fmt.Sprintf("%d", queuedMessageForSubscriber.ID), j)
		if err != nil {
			log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to add an unacked message")
		}
	}

	return b.store.Queue(fmt.Sprintf(clientMessageNotificationFmt, clientID)).Push(ctx, j)
}

// UpdateQueuedMessageForSubscriber updates a message in the message queue of a
// client
func (b *Broker) UpdateQueuedMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	j, err := encodeMessage(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Set(ctx, fmt.Sprintf("%d", queuedMessage.ID), j)
}

func (b *Broker) notifyClient(ctx context.Context, clientID string, c chan<- *QueuedMessage) {
	q := b.store.Queue(fmt.Sprintf(clientMessageNotificationFmt, clientID))

	for {
		result, err := q.Pop(ctx)
		if err != nil {
			break
		}

		queuedMessage, err := decodeMessage(result)
		if err != nil {
			continue
		}

		c <- queuedMessage
	}
}

// GetMessagesChannelForClient returns a channel containing messages for a
// client
func (b *Broker) GetMessagesChannelForClient(ctx context.Context, clientID string) <-chan *QueuedMessage {
	c := make(chan *QueuedMessage, 1)
	go b.notifyClient(ctx, clientID, c)
	return c
}

// ScanQueuedMessagesForClient iterates over the message queue of a client
func (b *Broker) ScanQueuedMessagesForClient(ctx context.Context, clientID string, f func(*QueuedMessage)) error {
	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Scan(ctx, func(ctx context.Context, k, v string) {
		queuedMessage, err := decodeMessage(v)
		if err != nil {
			return
		}

		f(queuedMessage)
	})
}

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
	"net"

	"github.com/dimkr/yodi/pkg/store"
	log "github.com/sirupsen/logrus"
)

type Broker struct {
	store store.Store
	ctx   context.Context
}

const (
	clientSet                    = "/clients"
	messageQueue                 = "/messages"
	topicSubscribersSetFmt       = "/topic/%s/subscribers"
	clientSubscriptionsSetFmt    = "/client/%s/subscriptions"
	clientMessageQueueFmt        = "/client/%s/messages"
	clientMessageNotificationFmt = "/client/%s/notify"
)

func NewBroker(ctx context.Context, store store.Store) (*Broker, error) {
	return &Broker{store: store, ctx: ctx}, nil
}

func (b *Broker) NewClient(conn net.Conn) (*Client, error) {
	return NewClient(b.ctx, conn, b)
}

func (b *Broker) AddClient(ctx context.Context, clientID string) error {
	return b.store.Set(clientSet).Add(ctx, clientID)
}

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

	if err := b.store.Queue(fmt.Sprintf(clientMessageQueueFmt, clientID)).Destroy(b.ctx); err != nil {
		return err
	}

	if err := b.store.Set(clientSet).Remove(b.ctx, clientID); err != nil {
		return err
	}

	return nil
}

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

func (b *Broker) Unsubscribe(ctx context.Context, clientID, topic string) error {
	if err := b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, topic)).Remove(ctx, clientID); err != nil {
		return err
	}

	return b.store.Set(fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Remove(ctx, topic)
}

func decodeMessage(raw []byte) (*QueuedMessage, error) {
	var msg QueuedMessage
	err := json.Unmarshal(raw, &msg)
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

	return decodeMessage([]byte(result))
}

func (b *Broker) PopQueuedMessage(ctx context.Context) (*QueuedMessage, error) {
	return b.popMessage(ctx, messageQueue)
}

func (b *Broker) QueueMessage(topic string, msg []byte, messageID uint16, qos QoS) error {
	queuedMessage := QueuedMessage{ID: messageID, Topic: topic, Message: string(msg), QoS: qos}

	log.WithFields(queuedMessage.LogFields()).Info("Queueing a message")

	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	err = b.store.Queue(messageQueue).Push(b.ctx, string(j))
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to queue a message")
	}

	return err
}

func (b *Broker) QueueMessageForSubscribers(queuedMessage *QueuedMessage) error {
	return b.store.Set(fmt.Sprintf(topicSubscribersSetFmt, queuedMessage.Topic)).Scan(b.ctx, func(ctx context.Context, clientID string) {
		b.QueueMessageForSubscriber(ctx, clientID, queuedMessage)
	})
}

func (b *Broker) UnqueueMessageForSubscriber(ctx context.Context, clientID string, messageID uint16) error {
	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Destroy(ctx)
}

func (b *Broker) QueueMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	if queuedMessage.QoS != QoS0 {
		err := b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Set(ctx, fmt.Sprintf("%d", queuedMessage.ID), string(j))
		if err != nil {
			log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to add an unacked message")
		}
	}

	return b.store.Queue(fmt.Sprintf(clientMessageNotificationFmt, clientID)).Push(ctx, string(j))
}

func (b *Broker) UpdateQueuedMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Set(ctx, fmt.Sprintf("%d", queuedMessage.ID), string(j))
}

func (b *Broker) notifyClient(ctx context.Context, clientID string, c chan<- *QueuedMessage) {
	key := fmt.Sprintf(clientMessageNotificationFmt, clientID)

	for {
		result, err := b.store.Queue(key).Pop(ctx)
		if err != nil {
			break
		}

		queuedMessage, err := decodeMessage([]byte(result))
		if err != nil {
			continue
		}

		c <- queuedMessage
	}
}

func (b *Broker) GetMessagesChannelForSubscriber(ctx context.Context, clientID string) <-chan *QueuedMessage {
	c := make(chan *QueuedMessage, 1)
	go b.notifyClient(ctx, clientID, c)
	return c
}

func (b *Broker) ScanQueuedMessagesForSubscriber(ctx context.Context, clientID string, f func(*QueuedMessage)) error {
	return b.store.Map(fmt.Sprintf(clientMessageQueueFmt, clientID)).Scan(ctx, func(ctx context.Context, k, v string) {
		queuedMessage, err := decodeMessage([]byte(v))
		if err != nil {
			return
		}

		f(queuedMessage)
	})
}

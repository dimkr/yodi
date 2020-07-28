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
	"time"

	log "github.com/sirupsen/logrus"
)

type QueuedMessage struct {
	ID        uint16    `json:"id"`
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	QoS       QoS       `json:"qos"`
	Duplicate bool      `json:"dup"`
	SendTime  time.Time `json:"ts"`
}

type Store interface {
	AddClient(ctx context.Context, clientID string) error
	RemoveClient(clientID string) error
	Subscribe(ctx context.Context, clientID, topic string) error
	Unsubscribe(ctx context.Context, clientID, topic string) error

	PopQueuedMessage(ctx context.Context) (*QueuedMessage, error)
	QueueMessage(topic string, msg []byte, messageID uint16, qos QoS) error

	QueueMessageForSubscribers(queuedMessage *QueuedMessage) error
	UnqueueMessageForSubscriber(ctx context.Context, clientID string, messageID uint16) error

	QueueMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error
	UpdateQueuedMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error

	GetMessagesChannelForSubscriber(ctx context.Context, clientID string) <-chan *QueuedMessage
	ScanQueuedMessagesForSubscriber(ctx context.Context, clientID string, f func(*QueuedMessage)) error
}

func (m *QueuedMessage) LogFields() log.Fields {
	return log.Fields{"id": m.ID, "topic": m.Topic}
}

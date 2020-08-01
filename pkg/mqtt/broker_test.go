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
	"testing"

	"github.com/dimkr/yodi/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestAddClient_UniqueClientID(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"

	assert.Nil(t, broker.AddClient(ctx, clientID))
	assert.NotNil(t, broker.AddClient(ctx, clientID))

	assert.Nil(t, broker.RemoveClient(clientID))
	assert.Nil(t, broker.AddClient(ctx, clientID))
}

func TestSubscribe_Twice(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"
	topic := "/topic"

	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))
	assert.NotNil(t, broker.Subscribe(ctx, clientID, topic))

	assert.Nil(t, broker.Unsubscribe(ctx, clientID, topic))
	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))
}

func TestSubscribe_TwoBrokers(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"
	topic := "/topic"

	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))
	assert.NotNil(t, broker.Subscribe(ctx, clientID, topic))

	otherBroker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	assert.NotNil(t, broker.Subscribe(ctx, clientID, topic))
	assert.NotNil(t, otherBroker.Subscribe(ctx, clientID, topic))

	assert.Nil(t, broker.Unsubscribe(ctx, clientID, topic))
	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))

	assert.Nil(t, broker.Unsubscribe(ctx, clientID, topic))
	assert.Nil(t, otherBroker.Subscribe(ctx, clientID, topic))

	assert.Nil(t, otherBroker.Unsubscribe(ctx, clientID, topic))
	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))
}

func TestQueueMessage_QoS0(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"
	topic := "/topic"
	msg := "{}"
	var ID uint16 = 1234

	assert.Nil(t, broker.QueueMessage(topic, msg, ID, QoS0))

	queuedMessage, err := broker.PopQueuedMessage(ctx)
	assert.Nil(t, err)
	assert.Equal(t, msg, queuedMessage.Message)

	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))

	c := broker.GetMessagesChannelForSubscriber(ctx, clientID)
	select {
	case <-c:
		t.FailNow()
	default:
	}

	assert.Nil(t, broker.QueueMessageForSubscribers(queuedMessage))

	assert.Equal(t, msg, (<-c).Message)

	queuedMessages := make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 0, len(queuedMessages))

	assert.NotNil(t, broker.UnqueueMessageForSubscriber(ctx, clientID, ID))
}

func TestQueueMessage_QoS1(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"
	topic := "/topic"
	msg := "{}"
	var ID uint16 = 1234

	assert.Nil(t, broker.QueueMessage(topic, msg, ID, QoS1))

	queuedMessage, err := broker.PopQueuedMessage(ctx)
	assert.Nil(t, err)
	assert.Equal(t, msg, queuedMessage.Message)

	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))

	c := broker.GetMessagesChannelForSubscriber(ctx, clientID)
	select {
	case <-c:
		t.FailNow()
	default:
	}

	assert.Nil(t, broker.QueueMessageForSubscribers(queuedMessage))

	assert.Equal(t, msg, (<-c).Message)

	queuedMessages := make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 1, len(queuedMessages))
	assert.Equal(t, msg, queuedMessages[0].Message)

	assert.Nil(t, broker.UnqueueMessageForSubscriber(ctx, clientID, ID))

	queuedMessages = make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 0, len(queuedMessages))
}

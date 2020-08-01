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

	assert.Nil(t, broker.QueueMessage(topic, msg, 1234, QoS0))

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

	receivedMessage := <-c
	assert.Equal(t, msg, receivedMessage.Message)

	queuedMessages := make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 0, len(queuedMessages))

	assert.NotNil(t, broker.UnqueueMessageForSubscriber(ctx, clientID, receivedMessage.ID))
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

	assert.Nil(t, broker.QueueMessage(topic, msg, 1234, QoS1))

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

	receivedMessage := <-c
	assert.Equal(t, msg, receivedMessage.Message)

	queuedMessages := make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 1, len(queuedMessages))
	assert.Equal(t, msg, queuedMessages[0].Message)

	assert.Nil(t, broker.UnqueueMessageForSubscriber(ctx, clientID, receivedMessage.ID))

	queuedMessages = make([]*QueuedMessage, 0)
	assert.Nil(t, broker.ScanQueuedMessagesForSubscriber(ctx, clientID, func(queuedMessage *QueuedMessage) {
		queuedMessages = append(queuedMessages, queuedMessage)
	}))

	assert.Equal(t, 0, len(queuedMessages))
}

func TestQueueMessage_QoS1_MessageIDReuse(t *testing.T) {
	store := store.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker, err := NewBroker(ctx, store)
	assert.Nil(t, err)

	clientID := "abcd"
	topic := "/topic"
	msg := "{}"

	assert.Nil(t, broker.QueueMessage(topic, msg, 1234, QoS1))
	assert.Nil(t, broker.QueueMessage(topic, msg, 1234, QoS1))

	assert.Nil(t, broker.Subscribe(ctx, clientID, topic))

	c := broker.GetMessagesChannelForSubscriber(ctx, clientID)
	select {
	case <-c:
		t.FailNow()
	default:
	}

	for i := 0; i < 2; i++ {
		queuedMessage, err := broker.PopQueuedMessage(ctx)
		assert.Nil(t, err)
		assert.Equal(t, msg, queuedMessage.Message)

		assert.Nil(t, broker.QueueMessageForSubscribers(queuedMessage))
	}

	receivedMessage := <-c
	assert.Equal(t, msg, receivedMessage.Message)

	secondReceivedMessage := <-c
	assert.Equal(t, msg, receivedMessage.Message)

	assert.NotEqual(t, secondReceivedMessage.ID, receivedMessage.ID)
}

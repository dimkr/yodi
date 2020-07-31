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

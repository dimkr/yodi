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
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dimkr/yodi/pkg/store"
)

type authenticator struct {
	store store.Store
}

// Authenticator authenticates MQTT clients
type Authenticator interface {
	AuthenticateUser(ctx context.Context, username, password string) (*User, error)
}

// ACL defines permissions and allowed QoS levels for topics
type ACL map[string]struct {
	Publish   bool `json:"publish,omitempty"`
	Subscribe bool `json:"subscribe,omitempty"`
	QoS       QoS  `json:"qos,omitempty"`
}

// User defines MQTT client credentials and permissions
type User struct {
	ACL      ACL    `json:"acl"`
	Password string `json:"password"`
}

const (
	usersMap = "/users"
)

// AuthenticatePublish determines whether or not a client is allowed to publish
// a message
func (a ACL) AuthenticatePublish(topic string, qos QoS) error {
	topicACL, ok := a[topic]
	if !ok {
		return errors.New("no ACL For topic")
	}

	if !topicACL.Publish {
		return errors.New("publishing is forbidden")
	}

	if qos <= topicACL.QoS {
		return fmt.Errorf("QoS level for %s is forbidden", topic)
	}

	return nil
}

// AuthenticateSubscribe determines whether or not a client is allowed to
// subscribe to a topic
func (a ACL) AuthenticateSubscribe(topic string, qos QoS) error {
	topicACL, ok := a[topic]
	if !ok {
		return errors.New("no ACL For topic")
	}

	if !topicACL.Subscribe {
		return errors.New("subscription is forbidden")
	}

	if qos <= topicACL.QoS {
		return fmt.Errorf("QoS level for %s is forbidden", topic)
	}

	return nil
}

func (a *authenticator) AuthenticateUser(ctx context.Context, username, password string) (*User, error) {
	j, err := a.store.Map(usersMap).Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("No such user '%s': %w", username, err)
	}

	var user User
	if err := json.Unmarshal([]byte(j), &user); err != nil {
		return nil, fmt.Errorf("Failed to find user '%s': %w", username, err)
	}

	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) != 1 {
		return nil, errors.New("bad password")
	}

	return &user, nil
}

// NewAuthenticator returns a new authenticator
func NewAuthenticator(store store.Store) Authenticator {
	return &authenticator{store: store}
}

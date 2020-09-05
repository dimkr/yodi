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
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
)

// SubscribeFixedHeader is the fixed header of a SUBSCRIBE control packet
type SubscribeFixedHeader struct {
	MessageID uint16
}

func (c *Client) authenticateSubscribe(topic string, qos QoS) error {
	log.WithFields(c.logFields).Info("Authenticating subscribe")
	return c.user.ACL.AuthenticateSubscribe(topic, qos)
}

func (c *Client) handleSubscribe(messageID uint16, topic string, qos QoS) error {
	if err := c.authenticateSubscribe(topic, qos); err != nil {
		return err
	}

	log.WithFields(c.logFields).Info("Subscribing to ", topic)

	if err := c.broker.Subscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	if err := c.writeSubscribeAck(messageID, qos); err != nil {
		return err
	}

	c.startMessagesRoutine.Do(func() {
		go c.queueMessages()
		go c.deliverMessages()
	})

	return nil
}

func (c *Client) readSubscribe() error {
	var subscribeFixedHeader SubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &subscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	buf := make([]byte, maxTopicLength)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}
	topic := string(buf[:n])

	qos := make([]byte, 1)
	n, err = c.reader.Read(qos)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("bad QoS")
	}

	return c.handleSubscribe(subscribeFixedHeader.MessageID, topic, QoS(qos[0]))
}

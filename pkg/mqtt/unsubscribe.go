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

// UnsubscribeFixedHeader is the fixed header of a UNSUBSCRIBE control packet
type UnsubscribeFixedHeader struct {
	MessageID uint16
}

func (c *Client) handleUnsubscribe(messageID uint16, topic string) error {
	log.WithFields(c.logFields).Info("unsubscribing from ", topic)

	if err := c.broker.Unsubscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	if err := c.writeUnsubscribeAck(messageID); err != nil {
		return err
	}

	return nil
}

func (c *Client) readUnsubscribe() error {
	var unsubscribeFixedHeader UnsubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &unsubscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	topic := make([]byte, maxTopicLength)
	n, err := stringReader.Read(topic)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}

	return c.handleUnsubscribe(unsubscribeFixedHeader.MessageID, string(topic[:n]))
}

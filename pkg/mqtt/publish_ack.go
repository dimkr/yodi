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

import "encoding/binary"

type PublishAckFixedHeader struct {
	MessageID uint16
}

func (c *Client) writePublishAck(messageID uint16) error {
	if err := c.writeFixedHeader(PublishAck, 2); err != nil {
		return err
	}

	hdr := PublishAckFixedHeader{MessageID: messageID}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	return nil
}

func (c *Client) handlePublishAck(messageID uint16) error {
	return c.store.UnqueueMessageForSubscriber(c.ctx, c.clientID, messageID)
}

func (c *Client) readPublishAck() error {
	var publishAckFixedHeader PublishAckFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &publishAckFixedHeader); err != nil {
		return err
	}

	return c.handlePublishAck(publishAckFixedHeader.MessageID)
}

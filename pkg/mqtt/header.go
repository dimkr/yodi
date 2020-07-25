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
	"errors"
	"fmt"
)

type Header struct {
	Flags         uint8
	MessageLength uint32
}

type MessageType uint8

const (
	Connect        MessageType = 0b0001
	ConnectAck     MessageType = 0b0010
	Disconnect     MessageType = 0b1110
	Subscribe      MessageType = 0b1000
	SubscribeAck   MessageType = 0b1001
	Unsubscribe    MessageType = 0b1010
	UnsubscribeAck MessageType = 0b1011
	Publish        MessageType = 0b0011
	PublishAck     MessageType = 0b0100
	PingRequest    MessageType = 0b1100
	PingResponse   MessageType = 0b1101

	qosMask       = 0b00000110
	qosShift      = 1
	duplicateFlag = 0b00001000

	ProtocolName    = "MQTT"
	ProtocolVersion = 4
)

func (h *Header) GetQoS() (QoS, error) {
	qos := QoS((h.Flags & qosMask) >> qosShift)

	if qos != QoS0 && qos != QoS1 {
		return QoS0, fmt.Errorf("invalid QoS level: %d", qos)
	}

	return qos, nil
}

func (c *Client) writeFixedHeaderWithFlags(messageType MessageType, messageLength int, qos QoS, duplicate bool) error {
	if messageLength > 16383 {
		return errors.New("Message is too long")
	}

	flags := (uint8(messageType) << 4) | (uint8(qos) << qosShift)
	if duplicate {
		flags |= duplicateFlag
	}
	hdr := append([]byte{flags}, encodeRemainingLength(uint32(messageLength))...)

	n, err := c.writer.Write(hdr)
	if err != nil {
		return err
	}
	if n != len(hdr) {
		return errors.New("Failed to write the entire header")
	}

	return nil
}

func (c *Client) writeFixedHeader(messageType MessageType, messageLength int) error {
	return c.writeFixedHeaderWithFlags(messageType, messageLength, 0, false)
}

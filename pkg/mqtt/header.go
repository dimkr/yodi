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

// Header is the fixed MQTT packet header
type Header struct {
	Flags         uint8
	MessageLength uint32
}

// MessageType is an MQTT packet type
type MessageType uint8

const (
	// Connect is a CONNECT control packet
	Connect MessageType = 0b0001

	// ConnectAck is a CONNACK control packet
	ConnectAck MessageType = 0b0010

	// Disconnect is a DISCONNECT control packet
	Disconnect MessageType = 0b1110

	// Subscribe is a SUBSCRIBE control packet
	Subscribe MessageType = 0b1000

	// SubscribeAck is a SUBACK control packet
	SubscribeAck MessageType = 0b1001

	// Unsubscribe is an UNSUBSCRIBE control packet
	Unsubscribe MessageType = 0b1010

	// UnsubscribeAck is an UNSUBACK control packet
	UnsubscribeAck MessageType = 0b1011

	// Publish is a PUBLISH control packet
	Publish MessageType = 0b0011

	// PublishAck is a PUBPACK control packet
	PublishAck MessageType = 0b0100

	// PingRequest is a PINGREQ control packet
	PingRequest MessageType = 0b1100

	// PingResponse is a PINGRESP control packet
	PingResponse MessageType = 0b1101

	qosMask       = 0b00000110
	qosShift      = 1
	duplicateFlag = 0b00001000

	// ProtocolName is the protocol name contained in a CONNECT control packet
	ProtocolName = "MQTT"

	// ProtocolVersion is the protocol version contained in a CONNECT control
	// packet
	ProtocolVersion = 4
)

// GetQoS returns the QoS level of an MQTT control packet
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

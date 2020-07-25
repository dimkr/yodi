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

	log "github.com/sirupsen/logrus"
)

type ReturnCode uint8

type ConnectAckFixedHeader struct {
	AckFlags   uint8
	ReturnCode ReturnCode
}

func (c *Client) writeConnectAck(code ReturnCode) error {
	if err := c.writeFixedHeader(ConnectAck, 2); err != nil {
		return err
	}

	hdr := ConnectAckFixedHeader{ReturnCode: code}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		log.Warn("failed to write connect ack")
		return err
	}

	return nil
}

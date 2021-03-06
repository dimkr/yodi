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
	"time"

	log "github.com/sirupsen/logrus"
)

const minPingInterval = time.Second * 20

func (c *Client) handlePing() error {
	log.WithFields(c.logFields).Debug("Responding to a ping")
	return c.writeFixedHeader(PingResponse, 0)
}

func (c *Client) readPing(hdr Header) error {
	if time.Now().Sub(c.lastPingTime) < minPingInterval {
		return errors.New("client pings too often")
	}

	if hdr.MessageLength != 0 {
		return errors.New("ping requests must have no payload")
	}

	return c.handlePing()
}

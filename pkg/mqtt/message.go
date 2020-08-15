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
	"time"

	log "github.com/sirupsen/logrus"
)

// QueuedMessage is a message published to a topic
type QueuedMessage struct {
	ID        uint16    `json:"id"`
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	QoS       QoS       `json:"qos"`
	Duplicate bool      `json:"dup"`
	SendTime  time.Time `json:"ts"`
}

// LogFields returns logging context for a message
func (m *QueuedMessage) LogFields() log.Fields {
	return log.Fields{"id": m.ID, "topic": m.Topic}
}

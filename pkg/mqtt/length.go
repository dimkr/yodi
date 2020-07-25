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
	"io"
)

func encodeRemainingLength(messageLength uint32) []uint8 {
	output := make([]uint8, 0)

	for i := 0; i < 4; i++ {
		encodedByte := uint8(messageLength % 128)

		messageLength = messageLength / 128

		if messageLength > 0 {
			encodedByte = encodedByte | 128
		}
		output = append(output, encodedByte)

		if messageLength == 0 {
			break
		}
	}

	return output
}

func decodeRemainingLength(reader io.Reader) (uint32, error) {
	var multiplier uint32 = 1
	var value uint32

	encodedByte := make([]byte, 1)

	for i := 0; i < 4; i++ {
		_, err := reader.Read(encodedByte)
		if err != nil {
			return 0, err
		}

		value += (uint32(encodedByte[0]) & 127) * multiplier

		multiplier *= 128
		if multiplier > 128*128*128 {
			return 0, errors.New("Malformed remaining length")
		}

		if encodedByte[0]&128 == 0 {
			break
		}
	}

	return value, nil
}

func (c *Client) readRemainingLength() (uint32, error) {
	return decodeRemainingLength(c.reader)
}

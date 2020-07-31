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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeRemainingLength(t *testing.T) {
	assert.Equal(t, []uint8{0}, encodeRemainingLength(0))
	assert.Equal(t, []uint8{1}, encodeRemainingLength(1))
	assert.Equal(t, []uint8{0x7f}, encodeRemainingLength(127))
	assert.Equal(t, []uint8{0x80, 0x01}, encodeRemainingLength(128))
	assert.Equal(t, []uint8{0xff, 0x7f}, encodeRemainingLength(16383))
	assert.Equal(t, []uint8{0xff, 0xff, 0x7f}, encodeRemainingLength(2097151))

	assert.Equal(t, []uint8{64}, encodeRemainingLength(64))
	assert.Equal(t, []uint8{193, 2}, encodeRemainingLength(321))
}

func TestDecodeRemainingLength(t *testing.T) {
	n, err := decodeRemainingLength(bytes.NewBuffer([]uint8{0}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(0), n)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{1}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), n)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0x7f}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(127), n)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0x80, 0x01}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(128), n)

	_, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0x80}))
	assert.NotNil(t, err)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0xff, 0x7f}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(16383), n)

	_, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0xff}))
	assert.NotNil(t, err)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{0xff, 0xff, 0x7f}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(2097151), n)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{64}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(64), n)

	n, err = decodeRemainingLength(bytes.NewBuffer([]uint8{193, 2}))
	assert.Nil(t, err)
	assert.Equal(t, uint32(321), n)

	_, err = decodeRemainingLength(bytes.NewBuffer([]uint8{193}))
	assert.NotNil(t, err)
}

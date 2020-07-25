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
	"io"
)

type StringReader struct {
	io.Reader
}

type StringWriter struct {
	io.Writer
}

func (w *StringWriter) Write(p []byte) (int, error) {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(p)))

	n, err := w.Writer.Write(length)
	if err != nil {
		return n, err
	}

	return w.Writer.Write(p)
}

func (r *StringReader) Read(p []byte) (int, error) {
	buf := make([]byte, 2)
	n, err := r.Reader.Read(buf)
	if err != nil {
		return n, err
	}
	if n != 2 {
		return 0, errors.New("failed to read length")
	}

	length := int(binary.BigEndian.Uint16(buf))
	if len(p) < length {
		return 0, errors.New("buffer is too small")
	}

	return r.Reader.Read(p[:length])
}

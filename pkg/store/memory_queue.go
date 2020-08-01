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

package store

import (
	"context"
)

type MemoryQueue struct {
	MemoryKey
	c chan string
}

const bufferSize = 64

func NewMemoryQueue(key MemoryKey) *MemoryQueue {
	return &MemoryQueue{c: make(chan string, bufferSize), MemoryKey: key}
}

func (q *MemoryQueue) Push(ctx context.Context, val string) error {
	q.c <- val
	return nil
}

func (q *MemoryQueue) Pop(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()

	case s := <-q.c:
		return s, nil
	}
}

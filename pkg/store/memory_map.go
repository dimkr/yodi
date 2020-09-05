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
	"fmt"
)

type memoryMap struct {
	*memoryKey
	items map[string]string
}

func newMemoryMap(key *memoryKey) *memoryMap {
	return &memoryMap{memoryKey: key, items: make(map[string]string)}
}

func (m *memoryMap) Get(ctx context.Context, k string) (string, error) {
	m.Lock()
	defer m.Unlock()

	v, ok := m.items[k]
	if !ok {
		return "", fmt.Errorf("%s does not exist", k)
	}

	return v, nil
}

func (m *memoryMap) Set(ctx context.Context, k, v string) error {
	m.Lock()
	defer m.Unlock()

	m.items[k] = v

	return nil
}

func (m *memoryMap) Remove(ctx context.Context, k string) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.items[k]; !ok {
		return fmt.Errorf("key does not exist in map")
	}

	delete(m.items, k)

	return nil
}

func (m *memoryMap) Scan(ctx context.Context, f func(context.Context, string, string)) error {
	m.Lock()
	defer m.Unlock()

	for k, v := range m.items {
		f(ctx, k, v)
	}

	return nil
}

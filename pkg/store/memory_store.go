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
	"errors"
	"sync"
)

type MemoryStore struct {
	lock  sync.Mutex
	items map[string]interface{}
}

func NewMemoryStore() Store {
	return &MemoryStore{items: make(map[string]interface{})}
}

func (s *MemoryStore) Lock() {
	s.lock.Lock()
}

func (s *MemoryStore) Unlock() {
	s.lock.Unlock()
}

func (s *MemoryStore) Set(key string) Set {
	s.Lock()
	defer s.Unlock()

	if set, ok := s.items[key]; ok {
		return set.(*MemorySet)
	}

	set := NewMemorySet(MemoryKey{store: s, key: key})
	s.items[key] = set
	return set
}

func (s *MemoryStore) Queue(key string) Queue {
	s.Lock()
	defer s.Unlock()

	if q, ok := s.items[key]; ok {
		return q.(*MemoryQueue)
	}

	q := NewMemoryQueue(MemoryKey{store: s, key: key})
	s.items[key] = q
	return q
}

func (s *MemoryStore) Map(key string) Map {
	s.Lock()
	defer s.Unlock()

	if m, ok := s.items[key]; ok {
		return m.(*MemoryMap)
	}

	m := NewMemoryMap(MemoryKey{store: s, key: key})
	s.items[key] = m
	return m
}

func (s *MemoryStore) Destroy(key string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.items[key]; !ok {
		return errors.New("key does not exist")
	}

	delete(s.items, key)
	return nil
}

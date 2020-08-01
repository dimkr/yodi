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

type MemorySet struct {
	MemoryKey
	items map[string]struct{}
}

func NewMemorySet(key MemoryKey) *MemorySet {
	return &MemorySet{MemoryKey: key, items: make(map[string]struct{})}
}

func (s *MemorySet) Add(ctx context.Context, val string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.items[val]; ok {
		return fmt.Errorf("value already exists in set")
	}

	s.items[val] = struct{}{}
	return nil
}

func (s *MemorySet) Remove(ctx context.Context, val string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.items[val]; !ok {
		return fmt.Errorf("value does not exist in set")
	}

	delete(s.items, val)

	return nil
}

func (s *MemorySet) Scan(ctx context.Context, f func(context.Context, string)) error {
	s.Lock()
	defer s.Unlock()

	for k := range s.items {
		f(ctx, k)
	}

	return nil
}

func (s *MemorySet) Members(ctx context.Context) ([]string, error) {
	s.Lock()
	defer s.Unlock()

	members := make([]string, 0)

	for k := range s.items {
		members = append(members, k)
	}

	return members, nil
}

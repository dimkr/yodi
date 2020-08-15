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
	"sync"
)

type memoryKey struct {
	lock  sync.Mutex
	store *memoryStore
	key   string
}

func (k *memoryKey) Destroy(ctx context.Context) error {
	return k.store.Destroy(k.key)
}

func (k *memoryKey) Lock() {
	k.lock.Lock()
}

func (k *memoryKey) Unlock() {
	k.lock.Unlock()
}

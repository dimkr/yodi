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

type RedisMap struct {
	RedisKey
}

func (m *RedisMap) Set(ctx context.Context, k, v string) error {
	_, err := m.Client.HSet(ctx, m.Key, k, v).Result()
	return err
}

func (m *RedisMap) Remove(ctx context.Context, k string) error {
	n, err := m.Client.HDel(ctx, m.Key, k).Result()
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("key does not exist in map")
	}

	return nil
}

func (m *RedisMap) Scan(ctx context.Context, f func(context.Context, string, string)) error {
	var cursor uint64
	var results []string
	var err error

	for {
		results, cursor, err = m.Client.HScan(ctx, m.Key, cursor, "*", 1).Result()
		if err != nil {
			return err
		}

		for i := 0; i < len(results); i += 2 {
			f(ctx, results[i], results[1])
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

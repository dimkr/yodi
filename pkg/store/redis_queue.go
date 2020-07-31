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

type RedisQueue struct {
	RedisKey
}

func (q *RedisQueue) Push(ctx context.Context, val string) error {
	_, err := q.Client.LPush(ctx, q.Key, val).Result()
	return err
}

func (q *RedisQueue) Pop(ctx context.Context) (string, error) {
	result, err := q.Client.BLPop(ctx, 0, q.Key).Result()
	return result[1], err
}

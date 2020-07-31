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
	"os"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	redisClient *redis.Client
	ctx         context.Context
}

func connectToRedis(ctx context.Context) (*redis.Client, error) {
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func NewRedisStore(ctx context.Context) (Store, error) {
	redisClient, err := connectToRedis(ctx)
	if err != nil {
		return nil, err
	}

	return &RedisStore{ctx: ctx, redisClient: redisClient}, nil
}

func (s *RedisStore) Set(key string) Set {
	return &RedisSet{RedisKey: RedisKey{Key: key, Client: s.redisClient}}
}

func (s *RedisStore) Queue(key string) Queue {
	return &RedisQueue{RedisKey: RedisKey{Key: key, Client: s.redisClient}}
}

func (s *RedisStore) Map(key string) Map {
	return &RedisMap{RedisKey: RedisKey{Key: key, Client: s.redisClient}}
}

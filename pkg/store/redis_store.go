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
	"runtime"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

var poolSize = 512 * runtime.NumCPU()

type redisStore struct {
	redisClient *redis.Client
}

func connectToRedis(ctx context.Context) (*redis.Client, error) {
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return nil, err
	}

	opts.PoolSize = poolSize

	// Heroku Redis runs Redis 5.0.9, which doesn't have the username argument
	for _, username := range []string{opts.Username, ""} {
		opts.Username = username
		client := redis.NewClient(opts)

		_, err = client.Ping(ctx).Result()
		if err != nil {
			log.WithError(err).Warn("Failed to connect to Redis")
			client.Close()
			continue
		}

		return client, nil
	}

	return nil, err
}

// NewRedisStore creates a Redis-backed store
func NewRedisStore(ctx context.Context) (Store, error) {
	redisClient, err := connectToRedis(ctx)
	if err != nil {
		return nil, err
	}

	return &redisStore{redisClient: redisClient}, nil
}

func (s *redisStore) Set(key string) Set {
	return &redisSet{redisKey: redisKey{Key: key, Client: s.redisClient}}
}

func (s *redisStore) Queue(key string) Queue {
	return &redisQueue{redisKey: redisKey{Key: key, Client: s.redisClient}}
}

func (s *redisStore) Map(key string) Map {
	return &redisMap{redisKey: redisKey{Key: key, Client: s.redisClient}}
}

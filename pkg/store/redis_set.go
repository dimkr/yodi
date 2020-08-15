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

type redisSet struct {
	redisKey
}

func (s *redisSet) Add(ctx context.Context, val string) error {
	n, err := s.Client.SAdd(ctx, s.Key, val).Result()
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("value already exists in set")
	}

	return nil
}

func (s *redisSet) Remove(ctx context.Context, val string) error {
	n, err := s.Client.SRem(ctx, s.Key, val).Result()
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("value does not exist in set")
	}

	return nil
}

func (s *redisSet) Scan(ctx context.Context, f func(context.Context, string)) error {
	var cursor uint64
	var values []string
	var err error

	for {
		values, cursor, err = s.Client.SScan(ctx, s.Key, cursor, "*", 1).Result()
		if err != nil {
			return err
		}

		for _, value := range values {
			f(ctx, value)
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *redisSet) Members(ctx context.Context) ([]string, error) {
	return s.Client.SMembers(ctx, s.Key).Result()
}

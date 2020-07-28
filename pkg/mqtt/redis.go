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

package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

type RedisStore struct {
	redisClient *redis.Client
}

const (
	clientSet                    = "/clients"
	messageQueue                 = "/messages"
	topicSubscribersSetFmt       = "/topic/%s/subscribers"
	clientSubscriptionsSetFmt    = "/client/%s/subscriptions"
	clientMessageQueueFmt        = "/client/%s/messages"
	clientMessageNotificationFmt = "/client/%s/notify"
)

func ConnectToRedis(ctx context.Context) (*redis.Client, error) {
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

func NewRedisStore(redisClient *redis.Client) Store {
	return &RedisStore{redisClient: redisClient}
}

func (s *RedisStore) AddClient(ctx context.Context, clientID string) error {
	n, err := s.redisClient.SAdd(ctx, clientSet, clientID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("client ID already in use: %s", clientID)
	}

	return nil
}

func (s *RedisStore) RemoveClient(clientID string) error {
	topics, err := s.redisClient.SMembers(context.Background(), fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Result()
	if err != nil {
		return err
	}

	for _, topic := range topics {
		s.redisClient.SRem(context.Background(), fmt.Sprintf(topicSubscribersSetFmt, topic), clientID)
	}

	if _, err := s.redisClient.Del(context.Background(), fmt.Sprintf(clientSubscriptionsSetFmt, clientID)).Result(); err != nil {
		return err
	}

	if _, err := s.redisClient.Del(context.Background(), fmt.Sprintf(clientMessageQueueFmt, clientID)).Result(); err != nil {
		return err
	}

	if _, err := s.redisClient.SRem(context.Background(), clientSet, clientID).Result(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStore) Subscribe(ctx context.Context, clientID, topic string) error {
	subscriptions := fmt.Sprintf(clientSubscriptionsSetFmt, clientID)

	n, err := s.redisClient.SAdd(ctx, subscriptions, topic).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		log.WithFields(log.Fields{"client_id": clientID, "topic": topic}).Debug("Client is already subscribed to topic")
		return nil
	}

	_, err = s.redisClient.SAdd(ctx, fmt.Sprintf(topicSubscribersSetFmt, topic), clientID).Result()
	if err != nil {
		s.redisClient.SRem(ctx, subscriptions, topic)
	}

	return err
}

func (s *RedisStore) Unsubscribe(ctx context.Context, clientID, topic string) error {
	n, err := s.redisClient.SRem(ctx, fmt.Sprintf(topicSubscribersSetFmt, topic), clientID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		log.WithFields(log.Fields{"client_id": clientID, "topic": topic}).Debug("Client was not subscribed to topic")
		return nil
	}

	_, err = s.redisClient.SRem(ctx, fmt.Sprintf(clientSubscriptionsSetFmt, clientID), topic).Result()
	return err
}

func decodeMessage(raw []byte) (*QueuedMessage, error) {
	var msg QueuedMessage
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		log.WithFields(log.Fields{"raw": raw}).WithError(err).Warn("failed to decode a message")
		return nil, err
	}

	return &msg, nil
}

func (s *RedisStore) popMessage(ctx context.Context, queue string) (*QueuedMessage, error) {
	result, err := s.redisClient.BLPop(ctx, 0, queue).Result()
	if err != nil {
		return nil, err
	}

	return decodeMessage([]byte(result[1]))
}

func (s *RedisStore) PopQueuedMessage(ctx context.Context) (*QueuedMessage, error) {
	return s.popMessage(ctx, messageQueue)
}

func (s *RedisStore) QueueMessage(topic string, msg []byte, messageID uint16, qos QoS) error {
	queuedMessage := QueuedMessage{ID: messageID, Topic: topic, Message: string(msg), QoS: qos}

	log.WithFields(queuedMessage.LogFields()).Info("Queueing a message")

	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	_, err = s.redisClient.LPush(context.Background(), messageQueue, j).Result()
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to queue a message")
	}

	return err
}

func (s *RedisStore) QueueMessageForSubscribers(queuedMessage *QueuedMessage) error {
	var cursor uint64
	var clientIDs []string
	var err error

	for {
		clientIDs, cursor, err = s.redisClient.SScan(context.Background(), fmt.Sprintf(topicSubscribersSetFmt, queuedMessage.Topic), cursor, "*", 1).Result()
		if err != nil {
			return err
		}

		for _, clientID := range clientIDs {
			if err := s.QueueMessageForSubscriber(context.Background(), clientID, queuedMessage); err != nil {
				continue
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *RedisStore) UnqueueMessageForSubscriber(ctx context.Context, clientID string, messageID uint16) error {
	n, err := s.redisClient.HDel(ctx, fmt.Sprintf(clientMessageQueueFmt, clientID), fmt.Sprintf("%d", messageID)).Result()
	if err != nil {
		return err
	}

	if n == 0 {
		return errors.New("Message is not queued")
	}

	return err
}

func (s *RedisStore) QueueMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	if queuedMessage.QoS != QoS0 {
		_, err = s.redisClient.HSet(ctx, fmt.Sprintf(clientMessageQueueFmt, clientID), fmt.Sprintf("%d", queuedMessage.ID), j).Result()
		if err != nil {
			log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to add an unacked message")
		}
	}

	_, err = s.redisClient.LPush(ctx, fmt.Sprintf(clientMessageNotificationFmt, clientID), j).Result()
	return err
}

func (s *RedisStore) UpdateQueuedMessageForSubscriber(ctx context.Context, clientID string, queuedMessage *QueuedMessage) error {
	j, err := json.Marshal(queuedMessage)
	if err != nil {
		log.WithFields(queuedMessage.LogFields()).WithError(err).Warn("failed to marshal a queued message")
		return err
	}

	_, err = s.redisClient.HSet(ctx, fmt.Sprintf(clientMessageQueueFmt, clientID), fmt.Sprintf("%d", queuedMessage.ID), j).Result()
	return err
}

func (s *RedisStore) notifyClient(ctx context.Context, clientID string, c chan<- *QueuedMessage) {
	key := fmt.Sprintf(clientMessageNotificationFmt, clientID)

	for {
		result, err := s.redisClient.BLPop(ctx, 0, key).Result()
		if err != nil {
			break
		}

		queuedMessage, err := decodeMessage([]byte(result[1]))
		if err != nil {
			continue
		}

		c <- queuedMessage
	}
}

func (s *RedisStore) GetMessagesChannelForSubscriber(ctx context.Context, clientID string) <-chan *QueuedMessage {
	c := make(chan *QueuedMessage, 1)
	go s.notifyClient(ctx, clientID, c)
	return c
}

func (s *RedisStore) ScanQueuedMessagesForSubscriber(ctx context.Context, clientID string, f func(*QueuedMessage)) error {
	var results []string
	var cursor uint64
	var err error

	for {
		results, cursor, err = s.redisClient.HScan(ctx, fmt.Sprintf(clientMessageQueueFmt, clientID), cursor, "", 1).Result()
		if err != nil {
			return err
		}

		for i := 1; i < len(results); i += 2 {
			queuedMessage, err := decodeMessage([]byte(results[i]))
			if err != nil {
				continue
			}

			f(queuedMessage)
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

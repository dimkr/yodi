package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

type Store struct {
	redisClient *redis.Client
}

type QueuedMessage struct {
	Topic   string `json:"topic"`
	Message string `json:"messge"`
}

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

func NewStore(redisClient *redis.Client) *Store {
	return &Store{redisClient: redisClient}
}

func (s *Store) AddClient(ctx context.Context, clientID string) error {
	n, err := s.redisClient.SAdd(ctx, "clients", clientID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("client ID already in use: %s", clientID)
	}

	return nil
}

func (s *Store) RemoveClient(clientID string) error {
	if _, err := s.redisClient.SRem(context.Background(), "clients", clientID).Result(); err != nil {
		return err
	}

	if _, err := s.redisClient.Del(context.Background(), "topics:"+clientID).Result(); err != nil {
		return err
	}

	if _, err := s.redisClient.Del(context.Background(), "messages:"+clientID).Result(); err != nil {
		return err
	}

	return nil
}

func (s *Store) Subscribe(ctx context.Context, clientID, topic string) error {
	n, err := s.redisClient.SAdd(ctx, "topics:"+clientID, topic).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		log.WithFields(log.Fields{"client_id": clientID, "topic": topic}).Warn("Client is already subscribed to topic")
	}

	return nil
}

func (s *Store) Unsubscribe(ctx context.Context, clientID, topic string) error {
	n, err := s.redisClient.SRem(ctx, "topics:"+clientID, topic).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		log.WithFields(log.Fields{"client_id": clientID, "topic": topic}).Warn("Client was not subscribed to topic")
	}

	return nil
}

func decodeMessage(raw []byte) (*QueuedMessage, error) {
	var msg QueuedMessage
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"raw": raw}).Warn("failed to decode a message")
		return nil, err
	}

	return &msg, nil
}

func (s *Store) popMessage(ctx context.Context, queue string) (*QueuedMessage, error) {
	result, err := s.redisClient.BLPop(ctx, 0, queue).Result()
	if err != nil {
		return nil, err
	}

	return decodeMessage([]byte(result[1]))
}

func (s *Store) PopQueuedMessage(ctx context.Context) (*QueuedMessage, error) {
	return s.popMessage(ctx, "messages")
}

func (s *Store) QueueMessage(topic string, msg []byte) error {
	j, err := json.Marshal(QueuedMessage{Topic: topic, Message: string(msg)})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"topic": topic, "msg": msg}).Warn("failed to marshal a queued message")
	}

	_, err = s.redisClient.LPush(context.Background(), "messages", j).Result()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"topic": topic, "msg": msg}).Warn("failed to queue a message")
	}

	return err
}

func (s *Store) PushMessage(queuedMessage *QueuedMessage) error {
	log.Info("Getting the list of client IDs")

	var cursor uint64
	var clientIDs []string
	var err error
	var subscribed bool

	for {
		clientIDs, cursor, err = s.redisClient.SScan(context.Background(), "clients", cursor, "*", 1).Result()
		if err != nil {
			return err
		}

		for _, clientID := range clientIDs {
			subscribed, err = s.redisClient.SIsMember(context.Background(), "topics:"+clientID, queuedMessage.Topic).Result()
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"client_id": clientID, "topic": queuedMessage.Topic}).Warn("failed to check whether or not client is subscribed to topic")
				continue
			}
			if !subscribed {
				continue
			}

			j, err := json.Marshal(queuedMessage)
			if err != nil {
				continue
			}

			log.WithFields(log.Fields{"client_id": clientID, "topic": queuedMessage.Topic}).Warn("Pushing message to client")

			_, err = s.redisClient.LPush(context.Background(), "messages:"+clientID, j).Result()
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"client_id": clientID, "topic": queuedMessage.Topic}).Warn("failed to push message to client")
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *Store) PopMessage(ctx context.Context, clientID string) (*QueuedMessage, error) {
	return s.popMessage(ctx, "messages:"+clientID)
}

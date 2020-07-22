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

const (
	clientSet                 = "/clients"
	messageQueue              = "/messages"
	topicSubscribersSetFmt    = "/topic/%s/subscribers"
	clientSubscriptionsSetFmt = "/client/%s/subscriptions"
	clientMessageQueueFmt     = "/client/%s/message_queue"
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

func NewStore(redisClient *redis.Client) *Store {
	return &Store{redisClient: redisClient}
}

func (s *Store) AddClient(ctx context.Context, clientID string) error {
	n, err := s.redisClient.SAdd(ctx, clientSet, clientID).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("client ID already in use: %s", clientID)
	}

	return nil
}

func (s *Store) RemoveClient(clientID string) error {
	if _, err := s.redisClient.SRem(context.Background(), clientSet, clientID).Result(); err != nil {
		return err
	}

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

	return nil
}

func (s *Store) Subscribe(ctx context.Context, clientID, topic string) error {
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

func (s *Store) Unsubscribe(ctx context.Context, clientID, topic string) error {
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
	return s.popMessage(ctx, messageQueue)
}

func (s *Store) QueueMessage(topic string, msg []byte) error {
	log.WithFields(log.Fields{"topic": topic, "message": string(msg)}).Info("Queueing a message")

	j, err := json.Marshal(QueuedMessage{Topic: topic, Message: string(msg)})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"topic": topic, "message": msg}).Warn("failed to marshal a queued message")
	}

	_, err = s.redisClient.LPush(context.Background(), messageQueue, j).Result()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"topic": topic, "message": msg}).Warn("failed to queue a message")
	}

	return err
}

func (s *Store) PushMessage(queuedMessage *QueuedMessage) error {
	var cursor uint64
	var clientIDs []string
	var err error

	for {
		clientIDs, cursor, err = s.redisClient.SScan(context.Background(), fmt.Sprintf(topicSubscribersSetFmt, queuedMessage.Topic), cursor, "*", 1).Result()
		if err != nil {
			return err
		}

		for _, clientID := range clientIDs {
			j, err := json.Marshal(queuedMessage)
			if err != nil {
				continue
			}

			log.WithFields(log.Fields{"client_id": clientID, "topic": queuedMessage.Topic, "message": queuedMessage.Message}).Debug("Pushing message to client")

			_, err = s.redisClient.LPush(context.Background(), fmt.Sprintf(clientMessageQueueFmt, clientID), j).Result()
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
	return s.popMessage(ctx, fmt.Sprintf(clientMessageQueueFmt, clientID))
}

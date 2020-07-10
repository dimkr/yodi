package mqtt

import (
	"context"
	"net"
)

type Broker struct {
	Store *Store
}

func NewBroker() (*Broker, error) {
	redisClient, err := ConnectToRedis(context.Background())
	if err != nil {
		return nil, err
	}

	return &Broker{Store: NewStore(redisClient)}, nil
}

func (b *Broker) NewClient(conn net.Conn) (*Client, error) {
	return NewClient(context.Background(), conn, b.Store)
}

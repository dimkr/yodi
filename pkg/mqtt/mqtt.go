package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Client struct {
	clientID             string
	logFields            log.Fields
	reader               io.Reader
	writer               io.Writer
	ctx                  context.Context
	cancel               context.CancelFunc
	startMessagesRoutine sync.Once
	registered           bool
	store                *Store
}

const (
	connectionTimeout = time.Hour

	maxTopicLength    = 64
	maxUsernameLength = 64
	maxPasswordLength = 64
)

var ErrDisconnected = errors.New("Client has disconnected")

func NewClient(parent context.Context, conn net.Conn, store *Store) (*Client, error) {
	t := time.Now().Add(connectionTimeout)
	if err := conn.SetDeadline(t); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithDeadline(parent, t)
	return &Client{
		logFields: log.Fields{},
		reader:    conn,
		writer:    conn,
		ctx:       ctx,
		cancel:    cancel,
		store:     store,
	}, nil
}

func (c *Client) Close() {
	if c.registered {
		if err := c.store.RemoveClient(c.clientID); err != nil {
			log.WithError(err).Warn("Failed to remove a client")
		}
	}

	c.cancel()
}

func (c *Client) deliverMessages() {
	log.WithFields(c.logFields).Info("Starting message delivery routine")

	for {
		queuedMessage, err := c.store.PopMessageForSubscriber(c.ctx, c.clientID)
		if err != nil {
			if c.ctx.Err() == nil {
				log.WithError(err).Warn("Failed to pop a message")
			}
			break
		}

		if err := c.publish(queuedMessage); err != nil {
			// if we failed to publish the message to the client, push it back to the queue
			c.store.QueueMessageForSubscriber(c.ctx, c.clientID, queuedMessage)
		}
	}

	log.WithFields(c.logFields).Info("Stopping message delivery routine")
}

func (c *Client) readPacket() error {
	flags := make([]byte, 1)
	if _, err := c.reader.Read(flags); err != nil {
		return err
	}

	length, err := c.readRemainingLength()
	if err != nil {
		return err
	}

	hdr := Header{Flags: flags[0], MessageLength: length}
	messageType := MessageType(hdr.Flags >> 4)

	switch messageType {
	case Publish:
		return c.readPublish(hdr)

	case PublishAck:
		return c.readPublishAck()

	case PingRequest:
		return c.readPing(hdr)

	case PingResponse:
		return c.readPingResponse(hdr)

	case Connect:
		return c.readConnect()

	case Subscribe:
		return c.readSubscribe()

	case Unsubscribe:
		return c.readUnsubscribe()

	case Disconnect:
		return ErrDisconnected

	default:
		return fmt.Errorf("unknown message type %d", messageType)
	}
}

func (c *Client) Run() error {
	for {
		if err := c.readPacket(); err != nil {
			if !errors.Is(err, ErrDisconnected) {
				log.WithFields(c.logFields).Warn(err)
			}
			return err
		}
	}
}

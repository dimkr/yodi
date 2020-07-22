package mqtt

import (
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
)

type QoS uint8

type SubscribeFixedHeader struct {
	MessageID uint16
}

func (c *Client) authenticateSubscribe(topic string, qos QoS) error {
	log.WithFields(c.logFields).Info("Authenticating subscribe")
	return nil
}

func (c *Client) handleSubscribe(messageID uint16, topic string, qos QoS) error {
	if err := c.authenticateSubscribe(topic, qos); err != nil {
		return err
	}

	log.WithFields(c.logFields).Info("Subscribing to ", topic)

	if err := c.store.Subscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	if err := c.writeSubscribeAck(messageID, qos); err != nil {
		return err
	}

	c.startMessagesRoutine.Do(func() {
		go c.deliverMessages()
	})

	return nil
}

func (c *Client) readSubscribe() error {
	var subscribeFixedHeader SubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &subscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	buf := make([]byte, maxTopicLength)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}
	topic := string(buf[:n])

	qos := make([]byte, 1)
	n, err = c.reader.Read(qos)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("bad QoS")
	}

	return c.handleSubscribe(subscribeFixedHeader.MessageID, topic, QoS(qos[0]))
}

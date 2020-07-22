package mqtt

import (
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
)

type UnsubscribeFixedHeader struct {
	MessageID uint16
}

func (c *Client) handleUnsubscribe(messageID uint16, topic string) error {
	log.WithFields(c.logFields).Info("unsubscribing from ", topic)

	if err := c.store.Unsubscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	if err := c.writeUnsubscribeAck(messageID); err != nil {
		return err
	}

	return nil
}

func (c *Client) readUnsubscribe() error {
	var unsubscribeFixedHeader UnsubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &unsubscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	topic := make([]byte, maxTopicLength)
	n, err := stringReader.Read(topic)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}

	return c.handleUnsubscribe(unsubscribeFixedHeader.MessageID, string(topic[:n]))
}

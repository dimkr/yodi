package mqtt

import (
	"errors"

	log "github.com/sirupsen/logrus"
)

func (c *Client) handlePublish(topic string, msg []byte) error {
	return c.store.QueueMessage(topic, msg)
}

func (c *Client) readPublish(hdr Header) error {
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

	if int(hdr.MessageLength) <= len(topic)+2 {
		return errors.New("no message")
	}

	buf = make([]byte, int(hdr.MessageLength)-len(topic)-2)
	total := 0
	for total < len(buf) {
		n, err = c.reader.Read(buf[total:])
		if err != nil {
			return err
		}
		if n == 0 {
			return errors.New("message is truncated")
		}
		total += n
	}

	return c.handlePublish(topic, buf)
}

func (c *Client) publish(topic string, msg []byte) error {
	log.WithFields(c.logFields).WithFields(log.Fields{"topic": topic, "msg": string(msg)}).Info("Passing a message")

	messageLength := 2 + len(topic) + len(msg)
	if messageLength > 255 {
		return errors.New("message is too big")
	}

	if err := c.writeFixedHeader(Publish, messageLength); err != nil {
		return err
	}

	stringWriter := StringWriter{c.writer}

	n, err := stringWriter.Write([]byte(topic))
	if err != nil {
		return err
	}
	if n != len(topic) {
		return errors.New("failed to send the topic")
	}

	n, err = c.writer.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("failed to send the message")
	}

	return nil
}

package mqtt

import (
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
)

func (c *Client) handlePublish(topic string, msg []byte, messageID uint16, qos QoS) error {
	if err := c.store.QueueMessage(topic, msg, messageID, qos); err != nil {
		return err
	}

	if qos == QoS0 {
		return nil
	}

	return c.writePublishAck(messageID)
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

	qos, err := hdr.GetQoS()
	if err != nil {
		return err
	}

	var messageID uint16
	headerSize := len(topic) + 2

	switch qos {
	case QoS0:

	case QoS1:
		if err := binary.Read(c.reader, binary.BigEndian, &messageID); err != nil {
			return err
		}

		headerSize += 2

	default:
		return errors.New("unknown QoS level")
	}

	if int(hdr.MessageLength) <= headerSize {
		return errors.New("no message")
	}

	buf = make([]byte, int(hdr.MessageLength)-headerSize)
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

	return c.handlePublish(topic, buf, messageID, qos)
}

func (c *Client) publish(queuedMessage *QueuedMessage) error {
	log.WithFields(c.logFields).WithFields(queuedMessage.LogFields()).Info("Delivering a message")

	messageLength := 2 + len(queuedMessage.Topic) + len(queuedMessage.Message)
	if queuedMessage.QoS == QoS1 {
		messageLength += 2
	}
	if messageLength > 255 {
		return errors.New("message is too big")
	}

	if err := c.writeFixedHeader(Publish, messageLength, queuedMessage.QoS); err != nil {
		return err
	}

	if queuedMessage.QoS == QoS1 {
		if err := binary.Write(c.writer, binary.BigEndian, &queuedMessage.ID); err != nil {
			return err
		}
	}

	stringWriter := StringWriter{c.writer}

	topic := []byte(queuedMessage.Topic)
	n, err := stringWriter.Write(topic)
	if err != nil {
		return err
	}
	if n != len(topic) {
		return errors.New("failed to send the topic")
	}

	msg := []byte(queuedMessage.Message)
	n, err = c.writer.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("failed to send the message")
	}

	return nil
}

package mqtt

import "encoding/binary"

type SubscribeAckFixedHeader struct {
	MessageID uint16
	QoS       QoS
}

func (c *Client) writeSubscribeAck(messageID uint16, qos QoS) error {
	if err := c.writeFixedHeader(SubscribeAck, 3, 0); err != nil {
		return err
	}

	hdr := SubscribeAckFixedHeader{MessageID: messageID, QoS: qos}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	return nil
}

package mqtt

import "encoding/binary"

type UnsubscribeAckFixedHeader struct {
	MessageID uint16
}

func (c *Client) writeUnsubscribeAck(messageID uint16) error {
	if err := c.writeFixedHeader(UnsubscribeAck, 2, 0); err != nil {
		return err
	}

	hdr := UnsubscribeAckFixedHeader{MessageID: messageID}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	return nil
}

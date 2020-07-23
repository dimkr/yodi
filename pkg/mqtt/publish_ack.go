package mqtt

import "encoding/binary"

type PublishAckFixedHeader struct {
	MessageID uint16
}

func (c *Client) writePublishAck(messageID uint16) error {
	if err := c.writeFixedHeader(PublishAck, 2, 0); err != nil {
		return err
	}

	hdr := PublishAckFixedHeader{MessageID: messageID}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	return nil
}

func (c *Client) handlePublishAck(messageID uint16) error {
	// TODO: do something here and re-senda message if not acked
	return nil
}

func (c *Client) readPublishAck() error {
	var publishAckFixedHeader PublishAckFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &publishAckFixedHeader); err != nil {
		return err
	}

	return c.handlePublishAck(publishAckFixedHeader.MessageID)
}

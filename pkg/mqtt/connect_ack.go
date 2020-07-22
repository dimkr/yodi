package mqtt

import (
	"encoding/binary"

	log "github.com/sirupsen/logrus"
)

type ReturnCode uint8

type ConnectAckFixedHeader struct {
	AckFlags   uint8
	ReturnCode ReturnCode
}

func (c *Client) writeConnectAck(code ReturnCode) error {
	if err := c.writeFixedHeader(ConnectAck, 2); err != nil {
		return err
	}

	hdr := ConnectAckFixedHeader{ReturnCode: code}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		log.Warn("failed to write connect ack")
		return err
	}

	return nil
}

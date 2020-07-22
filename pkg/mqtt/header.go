package mqtt

import "errors"

type Header struct {
	Flags         uint8
	MessageLength uint32
}

type MessageType uint8

const (
	Connect        MessageType = 0b0001
	ConnectAck     MessageType = 0b0010
	Disconnect     MessageType = 0b1110
	Subscribe      MessageType = 0b1000
	SubscribeAck   MessageType = 0b1001
	Unsubscribe    MessageType = 0b1010
	UnsubscribeAck MessageType = 0b1011
	Publish        MessageType = 0b0011
	PingRequest    MessageType = 0b1100
	PingResponse   MessageType = 0b1101

	ProtocolName    = "MQTT"
	ProtocolVersion = 4
)

func (c *Client) writeFixedHeader(messageType MessageType, messageLength int) error {
	if messageLength > 16383 {
		return errors.New("Message is too long")
	}

	hdr := append([]byte{uint8(messageType) << 4}, encodeRemainingLength(uint32(messageLength))...)

	n, err := c.writer.Write(hdr)
	if err != nil {
		return err
	}
	if n != len(hdr) {
		return errors.New("Failed to write the entire header")
	}

	return nil
}

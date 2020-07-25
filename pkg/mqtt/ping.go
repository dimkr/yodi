package mqtt

import (
	"errors"

	log "github.com/sirupsen/logrus"
)

func (c *Client) handlePing() error {
	log.WithFields(c.logFields).Debug("Responding to a ping")
	return c.writeFixedHeader(PingResponse, 0)
}

func (c *Client) readPing(hdr Header) error {
	if hdr.MessageLength != 0 {
		return errors.New("ping requests must have no payload")
	}

	return c.handlePing()
}

package mqtt

import "errors"

func (c *Client) readPingResponse(hdr Header) error {
	if hdr.MessageLength != 0 {
		return errors.New("ping responses must have no payload")
	}

	return nil
}

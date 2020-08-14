package mqtt

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type webSocketConn struct {
	*websocket.Conn
	frame io.Reader
}

func wrapWebSocket(conn *websocket.Conn) net.Conn {
	return &webSocketConn{Conn: conn}
}

func (c *webSocketConn) Read(b []byte) (int, error) {
	var err error
	var frameType, n int

	for n < len(b) {
		if c.frame == nil {
			frameType, c.frame, err = c.NextReader()
			if err != nil {
				return 0, err
			}

			if frameType != websocket.BinaryMessage {
				return 0, errors.New("unsupported message type")
			}
		}

		chunk, err := c.frame.Read(b)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return n, err
			}

			// start reading from the next frame once we're done reading from
			// the current one
			c.frame = nil
		}

		// TODO: is this check required?
		if chunk == 0 {
			c.frame = nil
		}

		n += chunk
	}

	return n, nil
}

func (c *webSocketConn) Write(b []byte) (n int, err error) {
	if err := c.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (c *webSocketConn) SetDeadline(t time.Time) error {
	// TODO
	return nil
}

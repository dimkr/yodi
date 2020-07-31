// This file is part of yodi.
//
// Copyright 2020 Dima Krasner
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mqtt

import (
	"encoding/binary"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type ConnectFlags uint8

type ConnectFixedHeader struct {
	ProtocolNameLength uint16
	ProtocolName       [4]byte
	ProtocolVersion    uint8
	ConnectFlags       ConnectFlags
	KeepAlive          uint16
}

const (
	ConnectionAccepted ReturnCode = iota
	ConnectionRefusedUnacceptableProtocolVersion
	ConnectionRefusedIdentifierRejected
	ConnectionRefusedServerUnavailable
	ConnectionRefusedBadUsernameOrPassword
	ConnectionRefusedNotAuthorized

	UsernameSet           ConnectFlags = 0b10000000
	PasswordSet           ConnectFlags = 0b01000000
	mandatoryConnectFlags              = UsernameSet | PasswordSet
)

func (c *Client) authenticateConnect(clientID, username, password string) error {
	log.WithFields(c.logFields).Info("Authenticating ", clientID, "@", username, "/", password)
	return nil
}

func (c *Client) handleConnect(clientID, username, password string) error {
	if err := c.authenticateConnect(clientID, username, password); err != nil {
		log.WithFields(c.logFields).Info("client has connected")
		c.writeConnectAck(ConnectionRefusedIdentifierRejected)
		return err
	}

	if err := c.broker.AddClient(c.ctx, clientID); err != nil {
		log.WithError(err).Warn("failed to add a client")
		c.writeConnectAck(ConnectionRefusedServerUnavailable)
		return err
	}
	c.registered = true

	if err := c.writeConnectAck(ConnectionAccepted); err != nil {
		log.Warn("failed to write connect ack")
		return err
	}

	c.clientID = clientID
	c.logFields["client_id"] = clientID

	log.WithFields(c.logFields).Info("client has connected")
	return nil
}

func (c *Client) readConnect(hdr Header) error {
	var connectFixedHeader ConnectFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &connectFixedHeader); err != nil {
		return err
	}

	proto := string(connectFixedHeader.ProtocolName[:])
	if proto != ProtocolName {
		return fmt.Errorf("Bad protocol name: %s", proto)
	}

	if connectFixedHeader.ProtocolVersion != ProtocolVersion {
		c.writeConnectAck(ConnectionRefusedUnacceptableProtocolVersion)
		return errors.New("Bad protocol version")
	}

	if connectFixedHeader.ConnectFlags&mandatoryConnectFlags != mandatoryConnectFlags {
		c.writeConnectAck(ConnectionRefusedNotAuthorized)
		return errors.New("Required connect flags are not set")
	}

	remainingLength := hdr.MessageLength
	if remainingLength <= 10+2 {
		return errors.New("Invalid Connect request")
	}
	remainingLength -= 10 + 2

	stringReader := StringReader{c.reader}

	buf := make([]byte, remainingLength)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty client ID")
	}
	clientID := string(buf[:n])

	remainingLength -= uint32(n)
	if remainingLength <= 2 {
		return errors.New("Invalid Connect request")
	}
	remainingLength -= 2

	buf = make([]byte, remainingLength)
	n, err = stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty username")
	}
	username := string(buf[:n])

	remainingLength -= uint32(n)
	if remainingLength <= 2 {
		return errors.New("Invalid Connect request")
	}
	remainingLength -= 2

	buf = make([]byte, remainingLength)
	n, err = stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty password")
	}
	password := string(buf[:n])

	if remainingLength != uint32(n) {
		return errors.New("Invalid Connect request")
	}

	return c.handleConnect(clientID, username, password)
}

package mqtt

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Header struct {
	Flags         uint8
	MessageLength uint32
}

type ConnectFlags uint8

type ConnectFixedHeader struct {
	ProtocolNameLength uint16
	ProtocolName       [4]byte
	ProtocolVersion    uint8
	ConnectFlags       ConnectFlags
	KeepAlive          uint16
}

type ReturnCode uint8

type ConnectAckFixedHeader struct {
	AckFlags   uint8
	ReturnCode ReturnCode
}

type SubscribeFixedHeader struct {
	MessageID uint16
}

type UnsubscribeFixedHeader struct {
	MessageID uint16
}

type QoS uint8

type SubscribeAckFixedHeader struct {
	MessageID uint16
	QoS       QoS
}

type UnsubscribeAckFixedHeader struct {
	MessageID uint16
}

type StringReader struct {
	io.Reader
}

type StringWriter struct {
	io.Writer
}

type Client struct {
	clientID             string
	logFields            log.Fields
	reader               io.Reader
	writer               io.Writer
	ctx                  context.Context
	cancel               context.CancelFunc
	startMessagesRoutine sync.Once
	registered           bool
	store                *Store
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

	UsernameSet ConnectFlags = 0b10000000
	PasswordSet ConnectFlags = 0b01000000

	mandatoryConnectFlags = UsernameSet | PasswordSet

	ConnectionAccepted ReturnCode = 0
	ProtocolName                  = "MQTT"
	ProtocolVersion               = 4

	connectionTimeout = time.Hour
)

var ErrDisconnected = errors.New("Client has disconnected")

func (w *StringWriter) Write(p []byte) (int, error) {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(p)))

	n, err := w.Writer.Write(length)
	if err != nil {
		return n, err
	}

	return w.Writer.Write(p)
}

func (r *StringReader) Read(p []byte) (int, error) {
	buf := make([]byte, 2)
	n, err := r.Reader.Read(buf)
	if err != nil {
		return n, err
	}
	if n != 2 {
		return 0, errors.New("failed to read length")
	}

	length := int(binary.BigEndian.Uint16(buf))
	if len(p) < length {
		return 0, errors.New("buffer is too small")
	}

	return r.Reader.Read(p[:length])
}

func NewClient(parent context.Context, conn net.Conn, store *Store) (*Client, error) {
	t := time.Now().Add(connectionTimeout)
	if err := conn.SetDeadline(t); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithDeadline(parent, t)
	return &Client{
		logFields: log.Fields{},
		reader:    conn,
		writer:    conn,
		ctx:       ctx,
		cancel:    cancel,
		store:     store,
	}, nil
}

func (c *Client) Close() {
	if c.registered {
		if err := c.store.RemoveClient(c.clientID); err != nil {
			log.WithError(err).Warn("Failed to remove a client")
		}
	}

	c.cancel()
}

func encodeRemainingLength(messageLength uint32) []uint8 {
	output := make([]uint8, 0)

	for i := 0; i < 4; i++ {
		encodedByte := uint8(messageLength % 128)

		messageLength = messageLength / 128

		if messageLength > 0 {
			encodedByte = encodedByte | 128
		}
		output = append(output, encodedByte)

		if messageLength == 0 {
			break
		}
	}

	return output
}

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

func (c *Client) publish(topic string, msg []byte) error {
	log.WithFields(c.logFields).WithFields(log.Fields{"topic": topic, "msg": string(msg)}).Info("Passing a message")

	messageLength := 2 + len(topic) + len(msg)
	if messageLength > 255 {
		return errors.New("message is too big")
	}

	if err := c.writeFixedHeader(Publish, messageLength); err != nil {
		return err
	}

	stringWriter := StringWriter{c.writer}

	n, err := stringWriter.Write([]byte(topic))
	if err != nil {
		return err
	}
	if n != len(topic) {
		return errors.New("failed to send the topic")
	}

	n, err = c.writer.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("failed to send the message")
	}

	return nil
}

func (c *Client) authenticateConnect(clientID, username, password string) error {
	log.WithFields(c.logFields).Info("Authenticating ", clientID, "@", username, "/", password)
	return nil
}

func (c *Client) handleConnect(clientID, username, password string) error {
	if err := c.authenticateConnect(clientID, username, password); err != nil {
		log.WithFields(c.logFields).Info("client has connected")
		return err
	}

	if err := c.store.AddClient(c.ctx, clientID); err != nil {
		log.WithError(err).Warn("failed to add a client")
		return err
	}
	c.registered = true

	c.writeFixedHeader(ConnectAck, 2)
	hdr := ConnectAckFixedHeader{ReturnCode: ConnectionAccepted}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		log.Warn("failed to write connect ack")
		return err
	}

	c.clientID = clientID
	c.logFields["client_id"] = clientID

	log.WithFields(c.logFields).Info("client has connected")
	return nil
}

func (c *Client) authenticateSubscribe(topic string, qos QoS) error {
	log.WithFields(c.logFields).Info("Authenticating subscribe")

	return nil

	if !strings.HasPrefix(topic, fmt.Sprintf("/%s/", c.clientID)) {
		return errors.New("topic name must begin with client ID")
	}

	return nil
}

func (c *Client) deliverMessages() {
	log.WithFields(c.logFields).Info("Starting message delivery routine")

	for {
		queuedMessage, err := c.store.PopMessage(c.ctx, c.clientID)
		if err != nil {
			if c.ctx.Err() == nil {
				log.WithError(err).Warn("Failed to pop a message")
			}
			break
		}

		c.publish(queuedMessage.Topic, []byte(queuedMessage.Message))
	}

	log.WithFields(c.logFields).Info("Stopping message delivery routine")
}

func (c *Client) handleSubscribe(messageID uint16, topic string, qos QoS) error {
	if err := c.authenticateSubscribe(topic, qos); err != nil {
		return err
	}

	log.WithFields(c.logFields).Info("Subscribing to ", topic)

	if err := c.store.Subscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	c.writeFixedHeader(SubscribeAck, 3)
	hdr := SubscribeAckFixedHeader{MessageID: messageID, QoS: qos}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	c.startMessagesRoutine.Do(func() {
		go c.deliverMessages()
	})

	return nil
}

func (c *Client) handleUnsubscribe(messageID uint16, topic string) error {
	log.WithFields(c.logFields).Info("unsubscribing from ", topic)

	if err := c.store.Unsubscribe(c.ctx, c.clientID, topic); err != nil {
		return err
	}

	c.writeFixedHeader(UnsubscribeAck, 2)
	hdr := UnsubscribeAckFixedHeader{MessageID: messageID}
	if err := binary.Write(c.writer, binary.BigEndian, &hdr); err != nil {
		return err
	}

	return nil
}

func (c *Client) handlePublish(topic string, msg []byte) error {
	log.WithFields(c.logFields).WithFields(log.Fields{"topic": topic, "msg": string(msg)}).Info("Pushing a message")
	return c.store.QueueMessage(topic, msg)
}

func (c *Client) readPublish(hdr Header) error {
	stringReader := StringReader{c.reader}

	buf := make([]byte, 64)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}
	topic := string(buf[:n])

	if int(hdr.MessageLength) <= len(topic)+2 {
		return errors.New("no message")
	}

	buf = make([]byte, int(hdr.MessageLength)-len(topic)-2)
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

	return c.handlePublish(topic, buf)
}

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

func (c *Client) readPingResponse(hdr Header) error {
	if hdr.MessageLength != 0 {
		return errors.New("ping responses must have no payload")
	}

	return nil
}

func (c *Client) readConnect() error {
	var connectFixedHeader ConnectFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &connectFixedHeader); err != nil {
		return err
	}

	proto := string(connectFixedHeader.ProtocolName[:])
	if proto != ProtocolName {
		return fmt.Errorf("Bad protocol name: %s", proto)
	}

	if connectFixedHeader.ProtocolVersion != ProtocolVersion {
		return errors.New("Bad protocol version")
	}

	if connectFixedHeader.ConnectFlags&mandatoryConnectFlags != mandatoryConnectFlags {
		return errors.New("Required connect flags are not set")
	}

	// TODO: validate hdr.MessageLength

	stringReader := StringReader{c.reader}

	buf := make([]byte, 64)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty client ID")
	}
	clientID := string(buf[:n])

	n, err = stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty username")
	}
	username := string(buf[:n])

	n, err = stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty password")
	}
	password := string(buf[:n])

	return c.handleConnect(clientID, username, password)
}

func (c *Client) readSubscribe() error {
	var subscribeFixedHeader SubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &subscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	buf := make([]byte, 64)
	n, err := stringReader.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}
	topic := string(buf[:n])

	qos := make([]byte, 1)
	n, err = c.reader.Read(qos)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("bad QoS")
	}

	return c.handleSubscribe(subscribeFixedHeader.MessageID, topic, QoS(qos[0]))
}

func (c *Client) readUnsubscribe() error {
	var unsubscribeFixedHeader UnsubscribeFixedHeader
	if err := binary.Read(c.reader, binary.BigEndian, &unsubscribeFixedHeader); err != nil {
		return nil
	}

	stringReader := StringReader{c.reader}

	topic := make([]byte, 64)
	n, err := stringReader.Read(topic)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty topic")
	}

	return c.handleUnsubscribe(unsubscribeFixedHeader.MessageID, string(topic[:n]))
}

func decodeRemainingLength(reader io.Reader) (uint32, error) {
	var multiplier uint32 = 1
	var value uint32

	encodedByte := make([]byte, 1)

	for i := 0; i < 4; i++ {
		_, err := reader.Read(encodedByte)
		if err != nil {
			return 0, err
		}

		value += (uint32(encodedByte[0]) & 127) * multiplier

		multiplier *= 128
		if multiplier > 128*128*128 {
			return 0, errors.New("Malformed remaining length")
		}

		if encodedByte[0]&128 == 0 {
			break
		}
	}

	return value, nil
}

func (c *Client) readRemainingLength() (uint32, error) {
	return decodeRemainingLength(c.reader)
}

func (c *Client) readPacket() error {
	flags := make([]byte, 1)
	if _, err := c.reader.Read(flags); err != nil {
		return err
	}

	length, err := c.readRemainingLength()
	if err != nil {
		return err
	}

	hdr := Header{Flags: flags[0], MessageLength: length}
	messageType := MessageType(hdr.Flags >> 4)

	switch messageType {
	case Publish:
		return c.readPublish(hdr)

	case PingRequest:
		return c.readPing(hdr)

	case PingResponse:
		return c.readPingResponse(hdr)

	case Connect:
		return c.readConnect()

	case Subscribe:
		return c.readSubscribe()

	case Unsubscribe:
		return c.readUnsubscribe()

	case Disconnect:
		return ErrDisconnected

	default:
		return fmt.Errorf("unknown message type %d", messageType)
	}
}

func (c *Client) Run() error {
	for {
		if err := c.readPacket(); err != nil {
			if !errors.Is(err, ErrDisconnected) {
				log.WithFields(c.logFields).Warn(err)
			}
			return err
		}
	}
}

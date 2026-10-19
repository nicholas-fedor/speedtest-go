// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
)

// loss is "LOSS", the command word that starts every packet-loss datagram.
var loss = []byte{0x4c, 0x4f, 0x53, 0x53}

// PacketLossSender handles UDP packet loss testing.
type PacketLossSender struct {
	// conn is the UDP connection to the server, or nil when not connected.
	conn net.Conn
	// dialer opens the connection.
	dialer *net.Dialer
	// ID is the client's UUID in upper case, under which the server counts the datagrams.
	ID string
	// host is the server's host:port.
	host string
	// raw is the datagram template, "LOSS <nonce> # <uuid>", whose # each datagram replaces with its index.
	raw []byte
	// nounce is a random number from 0 to 9999999999 that every datagram of the sender carries.
	nounce int64
	// withTimestamp marks datagrams that carry a timestamp. It is always false, because the sender sends none.
	withTimestamp bool
}

// NewPacketLossSender creates a new UDP packet loss sender.
//
// Parameters:
//   - uuid: the UUID the TCP client registered with INITPLOSS.
//   - dialer: opens the connection. A nil dialer makes [PacketLossSender.Connect] fail with [ErrNilDialer].
//
// Returns:
//   - *PacketLossSender: the sender, with no connection.
//   - error: the error from generating the nonce.
func NewPacketLossSender(uuid string, dialer *net.Dialer) (*PacketLossSender, error) {
	maxValue := int64(10000000000)
	b := big.NewInt(maxValue)

	n, err := rand.Int(rand.Reader, b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random nounce: %w", err)
	}

	nounce := n.Int64()
	p := &PacketLossSender{
		ID:            strings.ToUpper(uuid),
		nounce:        nounce,
		withTimestamp: false, // we close it as default, we won't be able to use it right now.
		dialer:        dialer,
	}
	p.raw = fmt.Appendf(nil, "%s %d %s %s", loss, nounce, "#", uuid)

	return p, nil
}

// Connect establishes a UDP connection to the specified host.
//
// Parameters:
//   - ctx: cancellation for the dial.
//   - host: the server's host:port.
//
// Returns:
//   - error: [ErrNilDialer] when the sender has no dialer, or the dial error.
func (ps *PacketLossSender) Connect(ctx context.Context, host string) error {
	if ps.dialer == nil {
		return ErrNilDialer
	}

	ps.host = host

	conn, err := ps.dialer.DialContext(ctx, "udp", ps.host)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %w", err)
	}

	ps.conn = conn

	return nil
}

// Close closes the UDP connection. It is safe to call before Connect and more than once.
//
// Returns:
//   - error: the error from closing the connection, if any.
func (ps *PacketLossSender) Close() error {
	if ps.conn == nil {
		return nil
	}

	err := ps.conn.Close()
	ps.conn = nil

	if err != nil {
		return fmt.Errorf("failed to close UDP connection: %w", err)
	}

	return nil
}

// Send sends a packet with the specified order value.
//
// Parameters:
//   - order: the datagram's index.
//
// Returns:
//   - error: [ErrEmptyConn] when not connected, or the write error.
func (ps *PacketLossSender) Send(order int) error {
	if ps.conn == nil {
		return ErrEmptyConn
	}

	payload := bytes.Replace(ps.raw, []byte{0x23}, []byte(strconv.Itoa(order)), 1)

	_, err := ps.conn.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write UDP packet: %w", err)
	}

	return nil
}

// generateUUID returns a random version 4 UUID.
//
// Returns:
//   - string: the UUID in its hyphenated lower-case form.
//   - error: the error from reading random bytes.
func generateUUID() (string, error) {
	randUUID := make([]byte, 16)

	_, err := rand.Read(randUUID)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for UUID: %w", err)
	}

	randUUID[8] = randUUID[8]&^0xc0 | 0x80
	randUUID[6] = randUUID[6]&^0xf0 | 0x40

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		randUUID[0:4],
		randUUID[4:6],
		randUUID[6:8],
		randUUID[8:10],
		randUUID[10:],
	), nil
}

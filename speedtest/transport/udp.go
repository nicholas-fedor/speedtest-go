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

var loss = []byte{0x4c, 0x4f, 0x53, 0x53}

// PacketLossSender handles UDP packet loss testing.
type PacketLossSender struct {
	ID            string   // UUID
	nounce        int64    // Random int (maybe) [0,10000000000)
	withTimestamp bool     // With timestamp (ten seconds level)
	conn          net.Conn // UDP Conn
	raw           []byte
	host          string
	dialer        *net.Dialer
}

// NewPacketLossSender creates a new UDP packet loss sender.
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
func (ps *PacketLossSender) Connect(ctx context.Context, host string) error {
	ps.host = host

	conn, err := ps.dialer.DialContext(ctx, "udp", ps.host)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %w", err)
	}

	ps.conn = conn

	return nil
}

// Send sends a packet with the specified order value.
func (ps *PacketLossSender) Send(order int) error {
	payload := bytes.Replace(ps.raw, []byte{0x23}, []byte(strconv.Itoa(order)), 1)

	_, err := ps.conn.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write UDP packet: %w", err)
	}

	return nil
}

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

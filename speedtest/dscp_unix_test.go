// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build unix && !aix

package speedtest

import (
	"context"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// testDSCP is the Expedited Forwarding code point, used as a recognizable mark.
const testDSCP = 46

// trafficClass reads a connected socket's IP_TOS or IPV6_TCLASS value.
func trafficClass(t *testing.T, conn net.Conn, family int) int {
	t.Helper()

	syscallConn, ok := conn.(syscall.Conn)
	require.True(t, ok)

	raw, err := syscallConn.SyscallConn()
	require.NoError(t, err)

	level, option := unix.IPPROTO_IP, unix.IP_TOS
	if family == 6 {
		level, option = unix.IPPROTO_IPV6, unix.IPV6_TCLASS
	}

	var (
		value  int
		getErr error
	)

	require.NoError(t, raw.Control(func(fd uintptr) {
		value, getErr = unix.GetsockoptInt(int(fd), level, option)
	}))
	require.NoError(t, getErr)

	return value
}

// loopbackListener listens on the loopback address of the given family, skipping the test when the host has none.
func loopbackListener(t *testing.T, family int) net.Listener {
	t.Helper()

	address := "127.0.0.1:0"
	if family == 6 {
		address = "[::1]:0"
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", address)
	if err != nil {
		t.Skipf("no IPv%d loopback: %v", family, err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

// TestMarkDSCP checks that TCP and UDP sockets of both IP versions carry the code point in the upper six bits of
// their traffic class.
func TestMarkDSCP(t *testing.T) {
	t.Parallel()

	for _, family := range []int{4, 6} {
		for _, network := range []string{"tcp", "udp"} {
			t.Run(network+string(rune('0'+family)), func(t *testing.T) {
				t.Parallel()

				listener := loopbackListener(t, family)
				dialer := &net.Dialer{Control: markDSCP(testDSCP)}

				conn, err := dialer.DialContext(
					context.Background(),
					network,
					listener.Addr().String(),
				)
				require.NoError(t, err)
				t.Cleanup(func() { _ = conn.Close() })

				assert.Equal(t, testDSCP<<2, trafficClass(t, conn, family))
			})
		}
	}
}

// TestMarkDSCPUnknownFamily checks that a network without an IP version fails instead of going out unmarked.
func TestMarkDSCPUnknownFamily(t *testing.T) {
	t.Parallel()

	err := markDSCP(testDSCP)("tcp", "127.0.0.1:80", nil)
	require.ErrorIs(t, err, errUnknownFamily)
}

// TestNewUserConfigDSCP checks that the client's TCP and UDP dialers mark their sockets, and that the user Control
// function runs after the marking.
func TestNewUserConfigDSCP(t *testing.T) {
	t.Parallel()

	listener := loopbackListener(t, 4)

	var seen []int

	client := New(WithUserConfig(&UserConfig{
		DSCP: testDSCP,
		DialerControl: func(_, _ string, raw syscall.RawConn) error {
			return raw.Control(func(fd uintptr) {
				value, _ := unix.GetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_TOS)
				seen = append(seen, value)
			})
		},
	}))

	tcpConn, err := client.tcpDialer.DialContext(
		context.Background(),
		"tcp",
		listener.Addr().String(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tcpConn.Close() })

	udpConn, err := client.udpDialer.DialContext(
		context.Background(),
		"udp",
		listener.Addr().String(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = udpConn.Close() })

	assert.Equal(t, testDSCP<<2, trafficClass(t, tcpConn, 4))
	assert.Equal(t, testDSCP<<2, trafficClass(t, udpConn, 4))
	assert.Equal(
		t,
		[]int{testDSCP << 2, testDSCP << 2},
		seen,
		"the user Control function runs on marked sockets",
	)
}

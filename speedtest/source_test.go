// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"errors"
	"net"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// loopbackInterface returns the name of an up loopback interface with an IPv4 address, which every supported
// platform has under a different name, skipping the test when there is none. It reads the addresses itself rather
// than through ResolveInterface, so the tests do not depend on the code they check.
func loopbackInterface(t *testing.T) string {
	t.Helper()

	interfaces, err := net.Interfaces()
	require.NoError(t, err)

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		require.NoError(t, err)

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil {
				return iface.Name
			}
		}
	}

	t.Skip("no loopback interface with an IPv4 address")

	return ""
}

// TestResolveInterface checks that an interface name resolves to its address, and that unknown names and IP
// literals are reported as not found.
func TestResolveInterface(t *testing.T) {
	t.Parallel()

	ip, err := ResolveInterface(loopbackInterface(t))
	require.NoError(t, err)
	assert.True(t, ip.IsLoopback())
	assert.NotNil(t, ip.To4(), "an IPv4 address is preferred")

	_, err = ResolveInterface("no-such-interface0")
	require.ErrorIs(t, err, ErrInterfaceNotFound)

	_, err = ResolveInterface("127.0.0.1")
	require.ErrorIs(t, err, ErrInterfaceNotFound)
}

// TestPickInterfaceAddress checks the address preference: IPv4 first, then IPv6 that is not link-local, and none
// when only link-local or non-IP addresses remain.
func TestPickInterfaceAddress(t *testing.T) {
	t.Parallel()

	ipv4 := &net.IPNet{IP: net.ParseIP("192.0.2.10").To4(), Mask: net.CIDRMask(24, 32)}
	global := &net.IPNet{IP: net.ParseIP("2001:db8::10"), Mask: net.CIDRMask(64, 128)}
	linkLocal := &net.IPNet{IP: net.ParseIP("fe80::10"), Mask: net.CIDRMask(64, 128)}
	other := &net.IPAddr{IP: net.ParseIP("192.0.2.20")}

	tests := []struct {
		name  string
		addrs []net.Addr
		want  net.IP
	}{
		{name: "ipv4 preferred", addrs: []net.Addr{linkLocal, global, ipv4}, want: ipv4.IP},
		{name: "ipv6 fallback", addrs: []net.Addr{linkLocal, global}, want: global.IP},
		{name: "link-local only", addrs: []net.Addr{linkLocal}, want: nil},
		{name: "no IP networks", addrs: []net.Addr{other}, want: nil},
		{name: "none", addrs: nil, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, pickInterfaceAddress(tt.addrs))
		})
	}
}

// TestResolveSourceInterface checks the binding for an interface name. Linux binds sockets to the device and sets
// no fixed address, so each dial gets an address of its destination's family. Other platforms set the address.
func TestResolveSourceInterface(t *testing.T) {
	t.Parallel()

	binding := resolveSource("tcp://" + loopbackInterface(t))

	if runtime.GOOS == "linux" {
		assert.NotNil(t, binding.control)
		assert.Nil(t, binding.tcp)
		assert.Nil(t, binding.ip)
		assert.Nil(t, binding.sourceIP)

		return
	}

	assert.Nil(t, binding.control)
	require.NotNil(t, binding.sourceIP)
	assert.True(t, binding.sourceIP.IsLoopback())
	assert.Equal(t, &net.TCPAddr{IP: binding.sourceIP}, binding.tcp)
	assert.Equal(t, &net.IPAddr{IP: binding.sourceIP}, binding.ip)
}

// TestPacketLossSourceIPInterface checks that packet loss accepts an interface name as its source.
func TestPacketLossSourceIPInterface(t *testing.T) {
	t.Parallel()

	ip := packetLossSourceIP("udp://" + loopbackInterface(t))
	require.NotNil(t, ip)
	assert.True(t, ip.IsLoopback())
}

// TestChainControl checks that nil entries are skipped, that the functions run in order, and that the chain stops
// at the first error.
func TestChainControl(t *testing.T) {
	t.Parallel()

	assert.Nil(t, chainControl(nil, nil))

	var calls []string

	record := func(name string, err error) dialControl {
		return func(string, string, syscall.RawConn) error {
			calls = append(calls, name)

			return err
		}
	}

	require.NoError(
		t,
		chainControl(record("device", nil), nil, record("user", nil))("tcp", "", nil),
	)
	assert.Equal(t, []string{"device", "user"}, calls)

	calls = nil
	failure := errors.New("bind failed")

	err := chainControl(record("device", failure), record("user", nil))("tcp", "", nil)
	require.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"device"}, calls, "a failure stops the chain")
}

// TestNewUserConfigSourceInterface checks that a client with an interface source reaches a server through it, and
// that a user Control function still runs alongside the device binding.
func TestNewUserConfigSourceInterface(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var userControls atomic.Int32

	client := New(WithUserConfig(&UserConfig{
		BaseURL: api.URL(),
		Source:  loopbackInterface(t),
		DialerControl: func(string, string, syscall.RawConn) error {
			userControls.Add(1)

			return nil
		},
	}))

	_, err := client.FetchUserInfoContext(context.Background())
	require.NoError(t, err)
	assert.Positive(t, userControls.Load(), "the user Control function should run")
}

// TestNewUserConfigSourceInterfaceDNS checks that with DNSBindSource, DNS sockets for an interface source run the
// client's whole Control chain, including the user's function.
func TestNewUserConfigSourceInterfaceDNS(t *testing.T) {
	t.Parallel()

	var userControls atomic.Int32

	client := New(WithUserConfig(&UserConfig{
		Source:        loopbackInterface(t),
		DNSBindSource: true,
		DialerControl: func(string, string, syscall.RawConn) error {
			userControls.Add(1)

			return nil
		},
	}))

	require.NotNil(t, client.tcpDialer.Resolver)

	conn, err := client.tcpDialer.Resolver.Dial(context.Background(), "udp", "127.0.0.1:53")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	assert.Positive(t, userControls.Load(), "DNS sockets should run the user Control function")
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build unix && !aix

package speedtest

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// dscpSupported reports whether this platform can mark sockets with a DSCP value.
const dscpSupported = true

// markDSCP returns a dialer Control function that sets a socket's traffic class to dscp.
//
// IPv4 sockets take IP_TOS and IPv6 sockets take IPV6_TCLASS. The code point fills the upper six bits of that byte,
// leaving the two ECN bits clear. A socket that cannot be marked fails its dial, so traffic never goes out unmarked.
//
// Parameters:
//   - dscp: the Differentiated Services code point, from 1 to 63.
//
// Returns:
//   - dialControl: the Control function.
func markDSCP(dscp int) dialControl {
	trafficClass := dscp << 2

	return func(network, _ string, conn syscall.RawConn) error {
		var level, option int

		switch networkFamily(network) {
		case 4:
			level, option = unix.IPPROTO_IP, unix.IP_TOS
		case 6:
			level, option = unix.IPPROTO_IPV6, unix.IPV6_TCLASS
		default:
			return fmt.Errorf("failed to set DSCP: %w %q", errUnknownFamily, network)
		}

		var setErr error

		err := conn.Control(func(fd uintptr) {
			setErr = unix.SetsockoptInt(int(fd), level, option, trafficClass)
		})
		if err != nil {
			return fmt.Errorf("failed to access socket to set DSCP: %w", err)
		}

		if setErr != nil {
			return fmt.Errorf("failed to set DSCP %d on %s socket: %w", dscp, network, setErr)
		}

		return nil
	}
}

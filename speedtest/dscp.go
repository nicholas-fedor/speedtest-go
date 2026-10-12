// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"errors"
	"fmt"
	"strings"
)

// maxDSCP is the largest Differentiated Services code point, which has six bits.
const maxDSCP = 63

// Errors for DSCP values.
var (
	// ErrInvalidDSCP is returned for a DSCP value outside 0 to 63.
	ErrInvalidDSCP = errors.New("DSCP must be from 0 to 63")
	// ErrDSCPUnsupported is returned for a nonzero DSCP value on a platform that cannot mark sockets.
	ErrDSCPUnsupported = errors.New("setting DSCP is not supported on this platform")
	// errUnknownFamily is returned when a dialer's network names no IP version.
	errUnknownFamily = errors.New("cannot tell the IP version of network")
)

// CheckDSCP reports whether a value can be used as [UserConfig.DSCP] on this platform.
//
// Parameters:
//   - dscp: the Differentiated Services code point, where 0 leaves sockets unmarked.
//
// Returns:
//   - error: an error wrapping [ErrInvalidDSCP] outside 0 to 63, [ErrDSCPUnsupported] for a nonzero value on a
//     platform that cannot mark sockets, or nil.
func CheckDSCP(dscp int) error {
	if dscp < 0 || dscp > maxDSCP {
		return fmt.Errorf("%w: %d", ErrInvalidDSCP, dscp)
	}

	if dscp != 0 && !dscpSupported {
		return ErrDSCPUnsupported
	}

	return nil
}

// dscpControl returns a dialer Control function that marks each socket with a DSCP value.
//
// The client has no way to report a configuration error yet, so an unusable value is logged and ignored.
//
// Parameters:
//   - dscp: the Differentiated Services code point.
//
// Returns:
//   - dialControl: the Control function, or nil when dscp is 0 or cannot be used.
func dscpControl(dscp int) dialControl {
	if dscp == 0 {
		return nil
	}

	err := CheckDSCP(dscp)
	if err != nil {
		dbg.Printf("Warning: skipping DSCP. err: %s\n", err.Error())

		return nil
	}

	return markDSCP(dscp)
}

// networkFamily returns the IP version of a network that a dialer passes to its Control function.
//
// Parameters:
//   - network: the network, such as "tcp4", "udp6", or "ip4:icmp".
//
// Returns:
//   - int: 4 or 6, or 0 when the network names no IP version.
func networkFamily(network string) int {
	base, _, _ := strings.Cut(network, ":")

	switch {
	case strings.HasSuffix(base, "4"):
		return 4
	case strings.HasSuffix(base, "6"):
		return 6
	default:
		return 0
	}
}

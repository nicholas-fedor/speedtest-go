// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build !unix || aix

package speedtest

// dscpSupported reports whether this platform can mark sockets with a DSCP value.
const dscpSupported = false

// markDSCP returns no Control function, since this platform cannot mark sockets.
//
// Parameters:
//   - dscp: the Differentiated Services code point, unused.
//
// Returns:
//   - dialControl: nil.
func markDSCP(_ int) dialControl {
	return nil
}

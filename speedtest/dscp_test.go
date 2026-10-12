// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckDSCP checks the range bounds and that nonzero values need a platform that can mark sockets.
func TestCheckDSCP(t *testing.T) {
	t.Parallel()

	for _, dscp := range []int{-1, 64} {
		require.ErrorIs(t, CheckDSCP(dscp), ErrInvalidDSCP, dscp)
	}

	require.NoError(t, CheckDSCP(0), "0 leaves sockets unmarked on every platform")

	for _, dscp := range []int{1, 46, 63} {
		if dscpSupported {
			require.NoError(t, CheckDSCP(dscp), dscp)
		} else {
			require.ErrorIs(t, CheckDSCP(dscp), ErrDSCPUnsupported, dscp)
		}
	}
}

// TestDSCPControl checks that unset and unusable values add no Control function.
func TestDSCPControl(t *testing.T) {
	t.Parallel()

	assert.Nil(t, dscpControl(0))
	assert.Nil(t, dscpControl(64))
	assert.Nil(t, dscpControl(-1))
	assert.Equal(t, dscpSupported, dscpControl(46) != nil)
}

// TestNetworkFamily checks the IP version read from each network a dialer can pass to its Control function.
func TestNetworkFamily(t *testing.T) {
	t.Parallel()

	tests := map[string]int{
		"tcp4":     4,
		"udp4":     4,
		"ip4:icmp": 4,
		"ip4":      4,
		"tcp6":     6,
		"udp6":     6,
		"ip6:58":   6,
		"tcp":      0,
		"unix":     0,
	}
	for network, want := range tests {
		assert.Equal(t, want, networkFamily(network), network)
	}
}

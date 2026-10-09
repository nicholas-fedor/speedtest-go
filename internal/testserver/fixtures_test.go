// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultUser checks that the default caller uses a documentation address that can never be real.
func TestDefaultUser(t *testing.T) {
	t.Parallel()

	user := DefaultUser()

	addr, err := netip.ParseAddr(user.IP)
	require.NoError(t, err)

	assert.True(
		t,
		netip.MustParsePrefix("203.0.113.0/24").Contains(addr),
		"the IP must be in TEST-NET-3",
	)
	assert.NotEmpty(t, user.ISP)
	assert.Equal(t, "JP", user.Country)
}

// TestDefaultServers checks that the defaults have distinct IDs and are listed farthest first from the default user.
func TestDefaultServers(t *testing.T) {
	t.Parallel()

	servers := DefaultServers()
	require.Len(t, servers, 2)

	assert.NotEqual(t, servers[0].ID, servers[1].ID)
	assert.Equal(
		t,
		"Osaka",
		servers[0].Name,
		"the farther server comes first so sorting by distance reorders them",
	)
	assert.Equal(t, "Tokyo", servers[1].Name)
}

// TestDefaultServers_ReturnsCopy checks that callers cannot change the defaults through the returned slice.
func TestDefaultServers_ReturnsCopy(t *testing.T) {
	t.Parallel()

	first := DefaultServers()
	first[0].ID = "changed"

	assert.Equal(t, "1002", DefaultServers()[0].ID)
}

// TestToXMLClient checks that every user field reaches the client element.
func TestToXMLClient(t *testing.T) {
	t.Parallel()

	user := User{IP: "203.0.113.9", Lat: "1.5", Lon: "2.5", ISP: "ISP", Country: "NZ"}

	assert.Equal(
		t,
		&xmlClient{IP: "203.0.113.9", Lat: "1.5", Lon: "2.5", ISP: "ISP", Country: "NZ"},
		toXMLClient(user),
	)
}

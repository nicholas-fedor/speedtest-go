// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// TestWriteCities checks that the list has its header and one line per city, sorted by label.
func TestWriteCities(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	require.NoError(t, writeCities(&buf))

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	cities := speedtest.Cities()

	require.Len(t, lines, len(cities)+2)
	assert.Equal(t, "Available city labels (case insensitive): ", lines[0])

	for i, city := range cities {
		assert.Contains(t, lines[i+2], "("+city.CC+")")
		assert.Contains(t, lines[i+2], city.Name)
	}
}

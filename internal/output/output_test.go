// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// TestWriteServerList_Exported checks that the exported writers show and mask the distance column.
func TestWriteServerList_Exported(t *testing.T) {
	t.Parallel()

	servers := speedtest.Servers{
		{
			ID:       "1",
			Name:     "Test Server",
			Sponsor:  "Test",
			Country:  "US",
			Distance: 10.0,
			Latency:  time.Millisecond,
		},
	}

	var plain, masked bytes.Buffer

	WriteServerList(&plain, servers)
	WriteMaskedServerList(&masked, servers)

	assert.Equal(t, "[    1]     10.00km 1ms \tTest Server (US) by Test \n", plain.String())
	assert.Equal(t, "[    1]     **.**km 1ms \tTest Server (US) by Test \n", masked.String())
}

// TestWriteAppInfo checks that the banner names the version between blank lines.
func TestWriteAppInfo(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	WriteAppInfo(&out)

	assert.Equal(t, "\n    "+version.Banner()+"\n\n", out.String())
}

// TestWriteServerList checks the server list lines, with the distance column shown or masked.
func TestWriteServerList(t *testing.T) {
	t.Parallel()

	servers := speedtest.Servers{
		{
			ID:       "6691",
			Distance: 9.03,
			Latency:  32 * time.Millisecond,
			Name:     "Shizuoka",
			Country:  "Japan",
			Sponsor:  "sudosan",
		},
		{
			ID:       "6087",
			Distance: 120.55,
			Latency:  -1,
			Name:     "Fussa-shi",
			Country:  "Japan",
			Sponsor:  "Allied",
		},
	}

	var plain, masked bytes.Buffer

	writeServerList(&plain, servers, formatDistance)
	writeServerList(
		&masked,
		servers,
		func(distance float64) string { return maskValue(formatDistance(distance)) },
	)

	assert.Equal(t, "[ 6691]      9.03km 32ms \tShizuoka (Japan) by sudosan \n"+
		"[ 6087]    120.55km Timeout \tFussa-shi (Japan) by Allied \n", plain.String())
	assert.Equal(t, "[ 6691]      *.**km 32ms \tShizuoka (Japan) by sudosan \n"+
		"[ 6087]    ***.**km Timeout \tFussa-shi (Japan) by Allied \n", masked.String())
}

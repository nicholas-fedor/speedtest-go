package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

func TestShowServerList(t *testing.T) {
	t.Parallel()

	type args struct {
		servers speedtest.Servers
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "show server list",
			args: args{
				servers: []*speedtest.Server{
					{ID: "1", Name: "Test Server", Sponsor: "Test", Country: "US", Distance: 10.0},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { ShowServerList(tt.args.servers) })
		})
	}
}

func TestAppInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"test app info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { AppInfo(false, false) })
			assert.NotPanics(t, func() { AppInfo(true, false) })
			assert.NotPanics(t, func() { AppInfo(false, true) })
		})
	}
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

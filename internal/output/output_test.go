package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
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
		})
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"test version returns non-empty string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Version()

			// Test that Version() returns a non-empty string
			assert.NotEmpty(t, got, "Version() should return a non-empty string")

			// Test that the output contains the stable sentinels
			assert.Contains(t, got, "dev", "Version() should contain commit")
			assert.Contains(t, got, "unknown", "Version() should contain build date")

			// Test that the output has the correct format (3 space-separated parts)
			parts := strings.Split(got, " ")
			assert.Len(t, parts, 3, "Version() should return format: 'version commit buildDate'")

			// Test that the version token has a "v" prefix
			assert.True(t, strings.HasPrefix(parts[0], "v"), "Version token should have v prefix")
			assert.NotEmpty(t, parts[0], "Version token should not be empty")
		})
	}
}

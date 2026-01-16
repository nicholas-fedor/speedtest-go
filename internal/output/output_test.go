package output

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/showwin/speedtest-go/speedtest"
)

func TestShowServerList(t *testing.T) {
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

package speedtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomServer(t *testing.T) {
	type args struct {
		host string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "valid host",
			args:    args{host: "example.com"},
			wantErr: false,
		},
		{
			name:    "empty host",
			args:    args{host: ""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := CustomServer(tt.args.host)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestSpeedtest_CustomServer(t *testing.T) {
	type args struct {
		host string
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			args:    args{host: "example.com"},
			wantErr: true,
		},
		{
			name:    "empty host",
			s:       &Speedtest{},
			args:    args{host: ""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.CustomServer(tt.args.host)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestServers_Available(t *testing.T) {
	tests := []struct {
		name    string
		servers Servers
	}{
		{
			name:    "empty servers",
			servers: Servers{},
		},
		{
			name:    "servers with available",
			servers: Servers{&Server{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.servers.Available()
			assert.NotNil(t, got)
		})
	}
}

func TestServers_Len(t *testing.T) {
	tests := []struct {
		name    string
		servers Servers
		want    int
	}{
		{
			name:    "empty servers",
			servers: Servers{},
			want:    0,
		},
		{
			name:    "single server",
			servers: Servers{&Server{}},
			want:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.servers.Len()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServers_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}

	tests := []struct {
		name    string
		servers Servers
		args    args
	}{
		{
			name:    "swap servers",
			servers: Servers{&Server{ID: "1"}, &Server{ID: "2"}},
			args:    args{i: 0, j: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			original := make(Servers, len(tt.servers))
			copy(original, tt.servers)
			tt.servers.Swap(tt.args.i, tt.args.j)
			// Verify swap occurred
			assert.Equal(t, original[tt.args.j], tt.servers[tt.args.i])
			assert.Equal(t, original[tt.args.i], tt.servers[tt.args.j])
		})
	}
}

func TestServers_Hosts(t *testing.T) {
	tests := []struct {
		name    string
		servers Servers
		want    []string
	}{
		{
			name:    "empty servers",
			servers: Servers{},
			want:    nil,
		},
		{
			name:    "single server",
			servers: Servers{&Server{Host: "example.com"}},
			want:    []string{"example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.servers.Hosts()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestByDistance_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}

	tests := []struct {
		name string
		b    ByDistance
		args args
		want bool
	}{
		{
			name: "first closer than second",
			b: ByDistance{
				Servers: Servers{
					{Distance: 100},
					{Distance: 200},
				},
			},
			args: args{i: 0, j: 1},
			want: true,
		},
		{
			name: "second closer than first",
			b: ByDistance{
				Servers: Servers{
					{Distance: 300},
					{Distance: 150},
				},
			},
			args: args{i: 0, j: 1},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.b.Less(tt.args.i, tt.args.j)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpeedtest_FetchServerByID(t *testing.T) {
	type args struct {
		serverID string
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			args:    args{serverID: "123"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.FetchServerByID(tt.args.serverID)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestFetchServerByID(t *testing.T) {
	type args struct {
		serverID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "invalid server ID",
			args:    args{serverID: "invalid"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FetchServerByID(tt.args.serverID)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestSpeedtest_FetchServerByIDContext(t *testing.T) {
	type args struct {
		serverID string
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			args:    args{serverID: "123"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			got, err := tt.s.FetchServerByIDContext(ctx, tt.args.serverID)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestSpeedtest_FetchServers(t *testing.T) {
	tests := []struct {
		name    string
		s       *Speedtest
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.FetchServers()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestFetchServers(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "fetch servers",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FetchServers()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestSpeedtest_FetchServerListContext(t *testing.T) {
	tests := []struct {
		name    string
		s       *Speedtest
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			got, err := tt.s.FetchServerListContext(ctx)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestFetchServerListContext(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "fetch server list with context",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			got, err := FetchServerListContext(ctx)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func Test_distance(t *testing.T) {
	type args struct {
		lat1 float64
		lon1 float64
		lat2 float64
		lon2 float64
	}

	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "same location",
			args: args{lat1: 0, lon1: 0, lat2: 0, lon2: 0},
			want: 0,
		},
		{
			name: "different locations",
			args: args{lat1: 0, lon1: 0, lat2: 1, lon2: 1},
			want: 157.425537108412, // Approximate distance in km
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := distance(tt.args.lat1, tt.args.lon1, tt.args.lat2, tt.args.lon2)
			assert.InDelta(t, tt.want, got, 0.1) // Allow small delta for floating point
		})
	}
}

func TestServers_FindServer(t *testing.T) {
	type args struct {
		serverID []int
	}

	tests := []struct {
		name    string
		servers Servers
		args    args
		wantErr bool
	}{
		{
			name:    "find server by ID",
			servers: Servers{&Server{ID: "123"}},
			args:    args{serverID: []int{123}},
			wantErr: false,
		},
		{
			name:    "no servers found",
			servers: Servers{},
			args:    args{serverID: []int{999}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.servers.FindServer(tt.args.serverID)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestServerList_String(t *testing.T) {
	tests := []struct {
		name    string
		servers ServerList
		want    string
	}{
		{
			name:    "empty server list",
			servers: ServerList{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.servers.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServers_String(t *testing.T) {
	tests := []struct {
		name    string
		servers Servers
		want    string
	}{
		{
			name:    "empty servers",
			servers: Servers{},
			want:    "",
		},
		{
			name:    "single server",
			servers: Servers{&Server{Host: "test.com", Country: "US"}},
			want:    "[    ] 0.00km  (US) by ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.servers.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServer_String(t *testing.T) {
	tests := []struct {
		name string
		s    *Server
		want string
	}{
		{
			name: "server string representation",
			s:    &Server{Host: "test.com", Country: "US"},
			want: "[    ] 0.00km  (US) by ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.s.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServer_CheckResultValid(t *testing.T) {
	tests := []struct {
		name string
		s    *Server
		want bool
	}{
		{
			name: "server with valid results",
			s:    &Server{DLSpeed: 100, ULSpeed: 50},
			want: true,
		},
		{
			name: "server with extreme speed ratio",
			s:    &Server{DLSpeed: 1000000, ULSpeed: 1}, // DL way faster than UL
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.s.CheckResultValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServer_testDurationTotalCount(t *testing.T) {
	tests := []struct {
		name string
		s    *Server
	}{
		{
			name: "nil server",
			s:    nil,
		},
		{
			name: "server with durations",
			s: &Server{
				TestDuration: TestDuration{
					Ping:     &[]time.Duration{time.Second}[0],
					Download: &[]time.Duration{time.Second}[0],
					Upload:   &[]time.Duration{time.Second}[0],
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.s != nil {
				tt.s.testDurationTotalCount()
			}
			// Test passes if no panic
		})
	}
}

func TestServer_getNotNullValue(t *testing.T) {
	type args struct {
		time *time.Duration
	}

	tests := []struct {
		name string
		s    *Server
		args args
		want time.Duration
	}{
		{
			name: "nil duration pointer",
			s:    &Server{},
			args: args{time: nil},
			want: 0,
		},
		{
			name: "valid duration pointer",
			s:    &Server{},
			args: args{time: &[]time.Duration{time.Second}[0]},
			want: time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.s.getNotNullValue(tt.args.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

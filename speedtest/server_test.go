package speedtest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

func TestCustomServer(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

// serverIDs returns the IDs of servers in order.
func serverIDs(servers Servers) []string {
	ids := make([]string, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, server.ID)
	}

	return ids
}

// TestSpeedtest_FetchServerByID checks the context-free wrapper against the fake API and with a nil client.
func TestSpeedtest_FetchServerByID(t *testing.T) {
	t.Parallel()

	_, err := (*Speedtest)(nil).FetchServerByID("1001")
	require.Error(t, err)

	api := testserver.NewAPI(t)

	server, err := newBaseURLClient(api.URL()).FetchServerByID("1001")
	require.NoError(t, err)
	assert.Equal(t, "1001", server.ID)
}

// TestSpeedtest_FetchServerByIDContext covers finding a server, computing its distance, and lookup failures.
func TestSpeedtest_FetchServerByIDContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		override *testserver.Response
		wantErr  error
		name     string
		id       string
		anyErr   bool
	}{
		{name: "known server", id: "1001"},
		{name: "unknown server", id: "9999", wantErr: ErrServerNotFound},
		{
			name:     "malformed XML",
			id:       "1001",
			override: &testserver.Response{Body: "<settings><servers"},
			anyErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			if tt.override != nil {
				api.SetResponse(testserver.PathServerLookup, *tt.override)
			}

			client := newBaseURLClient(api.URL())

			server, err := client.FetchServerByIDContext(context.Background(), tt.id)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.anyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.id, server.ID)
				assert.Equal(t, api.UploadURL(), server.URL)
				assert.Same(t, client, server.Context)
				assert.Positive(
					t,
					server.Distance,
					"distance comes from the client element in the lookup",
				)
			}
		})
	}

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		_, err := (*Speedtest)(nil).FetchServerByIDContext(context.Background(), "1001")
		require.Error(t, err)
	})
}

// TestSpeedtest_FetchServers checks the context-free wrapper against the fake API and with a nil client.
func TestSpeedtest_FetchServers(t *testing.T) {
	t.Parallel()

	_, err := (*Speedtest)(nil).FetchServers()
	require.Error(t, err)

	api := testserver.NewAPI(t)

	servers, err := newBaseURLClient(api.URL()).FetchServers()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(servers))
}

// TestSpeedtest_FetchServerListContext covers fetching, probing, and sorting the server list, and list failures.
func TestSpeedtest_FetchServerListContext(t *testing.T) {
	t.Parallel()

	t.Run("without user info keeps the API distance", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := newBaseURLClient(api.URL())

		servers, err := client.FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.Len(t, servers, 2)

		assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(servers))

		assertProbed(t, api, servers)

		for _, server := range servers {
			assert.Same(t, client, server.Context)
			assert.Zero(t, server.Distance, "distance needs the user's location")
		}
	})

	t.Run("with user info sorts by distance", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := newBaseURLClient(api.URL())

		_, err := client.FetchUserInfoContext(context.Background())
		require.NoError(t, err)

		servers, err := client.FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.Len(t, servers, 2)

		assert.Equal(
			t,
			[]string{"1001", "1002"},
			serverIDs(servers),
			"Tokyo is nearer to the Tokyo user",
		)
		assert.Less(t, servers[0].Distance, servers[1].Distance)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{Body: "[]"})

		_, err := newBaseURLClient(api.URL()).FetchServerListContext(context.Background())
		require.ErrorIs(t, err, ErrServerNotFound)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{Body: "[{"})

		_, err := newBaseURLClient(api.URL()).FetchServerListContext(context.Background())
		require.Error(t, err)
	})

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		_, err := (*Speedtest)(nil).FetchServerListContext(context.Background())
		require.Error(t, err)
	})
}

func Test_distance(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

// TestServerList_Encoding checks that the XML root name is used only for XML, so JSON output has no XMLName key.
func TestServerList_Encoding(t *testing.T) {
	t.Parallel()

	var list ServerList

	doc := `<settings><client ip="203.0.113.7"/><servers><server id="1001"/></servers></settings>`
	require.NoError(t, xml.Unmarshal([]byte(doc), &list))

	require.Len(t, list.Servers, 1)
	assert.Equal(t, "1001", list.Servers[0].ID)
	require.Len(t, list.Users, 1)
	assert.Equal(t, "203.0.113.7", list.Users[0].IP)

	data, err := json.Marshal(list)
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(data, &fields))

	assert.NotContains(t, fields, "XMLName")
	assert.Contains(t, fields, "servers")
	assert.Contains(t, fields, "users")
}

func TestServerList_String(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

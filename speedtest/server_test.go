package speedtest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

func TestClient_CustomServer(t *testing.T) {
	t.Parallel()

	type args struct {
		host string
	}

	tests := []struct {
		name    string
		s       *Client
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
			s:       &Client{},
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

// TestClient_FetchServerByID checks the context-free wrapper against the fake API and with a nil client.
func TestClient_FetchServerByID(t *testing.T) {
	t.Parallel()

	_, err := (*Client)(nil).FetchServerByID("1001")
	require.Error(t, err)

	api := testserver.NewAPI(t)

	server, err := newBaseURLClient(t, api.URL()).FetchServerByID("1001")
	require.NoError(t, err)
	assert.Equal(t, "1001", server.ID)
}

// TestClient_FetchServerByIDContext covers finding a server, computing its distance, and lookup failures.
func TestClient_FetchServerByIDContext(t *testing.T) {
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

			client := newBaseURLClient(t, api.URL())

			server, err := client.FetchServerByIDContext(context.Background(), tt.id)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.anyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.id, server.ID)
				assert.Equal(t, api.ServerUploadURL(tt.id), server.URL)
				assert.Equal(
					t,
					api.Host(),
					server.Host,
					"the lookup has no host, so it comes from the URL",
				)
				assert.Positive(
					t,
					server.Distance,
					"distance comes from the client element in the lookup",
				)
			}
		})
	}

	// The fake's user and server 1001 are both in Tokyo, so a distance of hundreds of kilometres shows that it was
	// measured from the Osaka location.
	t.Run("with a location measures from it", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		osaka, err := GetLocation("osaka")
		require.NoError(t, err)

		server, err := newTestClient(t, WithBaseURL(api.URL()), WithLocation(osaka)).
			FetchServerByIDContext(context.Background(), "1001")
		require.NoError(t, err)

		assert.InDelta(t, 395, server.Distance, 15)
	})

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		_, err := (*Client)(nil).FetchServerByIDContext(context.Background(), "1001")
		require.Error(t, err)
	})
}

// TestClient_FetchServers checks the context-free wrapper against the fake API and with a nil client.
func TestClient_FetchServers(t *testing.T) {
	t.Parallel()

	_, err := (*Client)(nil).FetchServers()
	require.Error(t, err)

	api := testserver.NewAPI(t)

	servers, err := newBaseURLClient(t, api.URL()).FetchServers()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(servers))
}

// TestClient_FetchServerListContext covers fetching, probing, and sorting the server list, and list failures.
func TestClient_FetchServerListContext(t *testing.T) {
	t.Parallel()

	t.Run("without a location keeps the API distance and order", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := newBaseURLClient(t, api.URL())

		servers, err := client.FetchServerListContext(context.Background())
		require.NoError(t, err)

		assert.Equal(
			t,
			[]string{"1002", "1001"},
			serverIDs(servers),
			"equal distances keep the listed order",
		)

		assertProbed(t, api, servers)

		for _, server := range servers {
			assert.Zero(t, server.Distance, "the fake reports no distance")
		}
	})

	// speedtest.net reports whole kilometres, so equal distances are common. They keep the listed order.
	t.Run(
		"without a location keeps reported distances and orders ties as listed",
		func(t *testing.T) {
			t.Parallel()

			listed := func(id string, distance int) testserver.Server {
				return testserver.Server{
					ID:       id,
					Name:     "Tokyo",
					Country:  "Japan",
					CC:       "JP",
					Sponsor:  "Example Networks",
					Lat:      "35.6895",
					Lon:      "139.6917",
					Distance: distance,
				}
			}

			api := testserver.NewAPI(
				t,
				testserver.WithServers(
					listed("2001", 5),
					listed("2002", 1),
					listed("2003", 5),
					listed("2004", 1),
				),
			)

			servers, err := newBaseURLClient(
				t,
				api.URL(),
			).FetchServerListContext(context.Background())
			require.NoError(t, err)

			assert.Equal(t, []string{"2002", "2004", "2001", "2003"}, serverIDs(servers))

			distances := make([]float64, 0, len(servers))
			for _, server := range servers {
				distances = append(distances, server.Distance)
			}

			assert.Equal(t, []float64{1, 1, 5, 5}, distances)
		},
	)

	// The fake's user is in Tokyo, so measuring from Osaka proves the distances come from the location.
	for _, tt := range []struct {
		city string
		want []string
	}{
		{city: "tokyo", want: []string{"1001", "1002"}},
		{city: "osaka", want: []string{"1002", "1001"}},
	} {
		t.Run("with a location sorts by distance from "+tt.city, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			location, err := GetLocation(tt.city)
			require.NoError(t, err)

			servers, err := newTestClient(t, WithBaseURL(api.URL()), WithLocation(location)).
				FetchServerListContext(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tt.want, serverIDs(servers))
			assert.Less(t, servers[0].Distance, servers[1].Distance)
			assert.Less(t, servers[0].Distance, 20.0, "the nearest server is in the same city")
		})
	}

	t.Run("with a location measures the alternative list", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{})

		location, err := GetLocation("tokyo")
		require.NoError(t, err)

		servers, err := newTestClient(t, WithBaseURL(api.URL()), WithLocation(location)).
			FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.True(t, api.Requested(testserver.PathServersStatic))

		assert.Equal(t, []string{"1001", "1002"}, serverIDs(servers))
		assert.Positive(t, servers[1].Distance)
	})

	// The alternative list has no distances, so a server that cannot be measured would otherwise rank first at 0 km.
	t.Run("with a location ranks servers without coordinates last", func(t *testing.T) {
		t.Parallel()

		unplaced := testserver.DefaultServers()[0]
		unplaced.ID = "3001"
		unplaced.Lat = "?"

		listed := append([]testserver.Server{unplaced}, testserver.DefaultServers()...)
		api := testserver.NewAPI(t, testserver.WithServers(listed...))
		api.SetResponse(testserver.PathServers, testserver.Response{})

		location, err := GetLocation("tokyo")
		require.NoError(t, err)

		servers, err := newTestClient(t, WithBaseURL(api.URL()), WithLocation(location)).
			FetchServerListContext(context.Background())
		require.NoError(t, err)

		assert.Equal(t, []string{"1001", "1002", "3001"}, serverIDs(servers))
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{Body: "[]"})

		_, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.ErrorIs(t, err, ErrServerNotFound)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{Body: "[{"})

		_, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.Error(t, err)
	})

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		_, err := (*Client)(nil).FetchServerListContext(context.Background())
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

// TestFetchServerListContext_TCPPingMode checks that TCP ping mode probes every listed server over TCP.
func TestFetchServerListContext_TCPPingMode(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)
	api := testserver.NewAPI(t, testserver.WithServerHost(tcp.Addr()))
	client := newTestClient(t, WithBaseURL(api.URL()), WithPingMode(TCP))

	servers, err := client.FetchServerListContext(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, 2)

	for _, server := range servers {
		assert.NotEqual(
			t,
			time.Duration(PingTimeout),
			server.Latency,
			"server %s probe timed out",
			server.ID,
		)
	}

	assert.GreaterOrEqual(t, tcp.Accepted(), len(servers), "each server should be pinged over TCP")
	assert.False(t, api.Requested(testserver.PathLatency), "TCP mode must not use HTTP probes")
}

// TestFetchServerListContext_ErrorStatus checks how the server list handles error statuses: a failing JSON list
// falls back to the XML list, a failing XML list is an error, and servers whose latency probe fails are marked as
// timed out so that they cannot be selected.
func TestFetchServerListContext_ErrorStatus(t *testing.T) {
	t.Parallel()

	failure := testserver.Response{Status: http.StatusServiceUnavailable, Body: "unavailable"}

	t.Run("JSON list fails", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, failure)

		servers, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.NoError(t, err)

		assert.Len(t, servers, 2)
		assert.True(t, api.Requested(testserver.PathServersStatic))
	})

	t.Run("both lists fail", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, failure)
		api.SetResponse(testserver.PathServersStatic, failure)

		_, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.ErrorIs(t, err, ErrUnexpectedStatus)
	})

	t.Run("latency probes fail", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathLatency, failure)

		servers, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.Len(t, servers, 2)

		for _, server := range servers {
			assert.Equal(
				t,
				time.Duration(PingTimeout),
				server.Latency,
				"server %s should be unreachable",
				server.ID,
			)
		}

		assert.Empty(t, *servers.Available(), "no failing server may be selected")
	})
}

// TestFetchServerByIDContext_ErrorStatus checks that a failing server lookup is reported as a status error.
func TestFetchServerByIDContext_ErrorStatus(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(testserver.PathServerLookup, testserver.Response{Status: http.StatusNotFound})

	_, err := newBaseURLClient(t, api.URL()).FetchServerByIDContext(context.Background(), "1001")
	require.ErrorIs(t, err, ErrUnexpectedStatus)
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

// TestServers_FilterCountry checks that codes match without regard to case, that the order is kept, and that no
// match gives an empty list.
func TestServers_FilterCountry(t *testing.T) {
	t.Parallel()

	servers := Servers{
		{ID: "1", CC: "GB"},
		{ID: "2", CC: "JP"},
		{ID: "3", CC: "gb"},
		{ID: "4", CC: "US"},
	}

	ids := func(filtered Servers) []string {
		result := make([]string, 0, len(filtered))
		for _, server := range filtered {
			result = append(result, server.ID)
		}

		return result
	}

	assert.Equal(t, []string{"1", "3"}, ids(servers.FilterCountry("gb")))
	assert.Equal(t, []string{"1", "3", "4"}, ids(servers.FilterCountry("US", "GB")))
	assert.Empty(t, servers.FilterCountry("FR"))
	assert.Empty(t, servers.FilterCountry())
}

// TestServerCountryCode checks that servers keep their country code from the JSON list and the XML list. A lookup by
// ID has none to keep.
func TestServerCountryCode(t *testing.T) {
	t.Parallel()

	t.Run("JSON list", func(t *testing.T) {
		t.Parallel()

		servers, err := newBaseURLClient(
			t,
			testserver.NewAPI(t).URL(),
		).FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.NotEmpty(t, servers)

		for _, server := range servers {
			assert.Equal(t, "JP", server.CC)
		}
	})

	t.Run("XML list", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(testserver.PathServers, testserver.Response{})

		servers, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
		require.NoError(t, err)
		require.True(
			t,
			api.Requested(testserver.PathServersStatic),
			"the test must use the XML list",
		)
		require.NotEmpty(t, servers)

		for _, server := range servers {
			assert.Equal(t, "JP", server.CC)
		}
	})
}

// TestLondonCountryCode checks that London uses the ISO code speedtest.net reports, GB, not UK.
func TestLondonCountryCode(t *testing.T) {
	t.Parallel()

	location, err := GetLocation("london")
	require.NoError(t, err)
	assert.Equal(t, "gb", location.CC)
}

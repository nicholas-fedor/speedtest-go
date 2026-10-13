package speedtest

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// mirrorPrefix is the path prefix the fake API is mounted under, so tests prove base paths survive.
const mirrorPrefix = "/mirror"

// newBaseURLClient returns a client that talks to the given base URL.
//
// Parameters:
//   - t: the test, which fails when the base URL is invalid.
//   - baseURL: the base URL.
//
// Returns:
//   - *Client: the client.
func newBaseURLClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	return newTestClient(t, WithBaseURL(baseURL))
}

// newMirrorAPI starts a fake API whose speedtest.net endpoints are served under mirrorPrefix.
func newMirrorAPI(t *testing.T) *testserver.API {
	t.Helper()

	return testserver.NewAPI(t, testserver.WithPathPrefix(mirrorPrefix))
}

// requestFor returns the first recorded request for endpoint, failing the test when there is none.
func requestFor(t *testing.T, api *testserver.API, endpoint string) testserver.Request {
	t.Helper()

	for _, req := range api.Requests() {
		if req.Endpoint == endpoint {
			return req
		}
	}

	require.Failf(t, "endpoint not requested", "no request for %s", endpoint)

	return testserver.Request{}
}

// assertProbed checks that every server was sent a latency probe and recorded a latency that is not a timeout.
//
// Whether a server was probed comes from the fake API's per-server request log, not from the latency value. A
// loopback round trip can finish between ticks of a coarse clock, such as the monotonic clock on Windows, and then
// measures exactly zero, which is also the value of a latency that was never set. A failed probe is PingTimeout.
func assertProbed(t *testing.T, api *testserver.API, servers Servers) {
	t.Helper()

	probed := map[string]bool{}

	for _, req := range api.Requests() {
		if req.Endpoint == testserver.PathLatency && req.ServerID != "" {
			probed[req.ServerID] = true
		}
	}

	timeout := time.Duration(PingTimeout)

	for _, server := range servers {
		assert.True(t, probed[server.ID], "server %s should receive a latency probe", server.ID)
		assert.NotEqual(t, timeout, server.Latency, "server %s probe timed out", server.ID)
		assert.GreaterOrEqual(t, server.Latency, time.Duration(0), "server %s latency", server.ID)
	}
}

// TestParseBaseURL covers the accepted base URL forms and each rejection rule.
func TestParseBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "default", raw: DefaultBaseURL, want: DefaultBaseURL},
		{name: "plain http", raw: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{
			name: "path prefix",
			raw:  "https://mirror.example/speedtest",
			want: "https://mirror.example/speedtest",
		},
		{name: "uppercase scheme", raw: "HTTPS://mirror.example", want: "https://mirror.example"},
		{name: "empty", raw: "", wantErr: true},
		{name: "relative", raw: "www.speedtest.net", wantErr: true},
		{name: "unsupported scheme", raw: "ftp://mirror.example", wantErr: true},
		{name: "missing host", raw: "https://", wantErr: true},
		{name: "query", raw: "https://mirror.example?key=value", wantErr: true},
		{name: "fragment", raw: "https://mirror.example#top", wantErr: true},
		{name: "unparsable", raw: "https://mirror.example/%zz", wantErr: true},
		{name: "username", raw: "https://user@mirror.example", wantErr: true},
		{name: "username and password", raw: "https://user:secret@mirror.example", wantErr: true},
		{
			name:    "credentials with unsupported scheme",
			raw:     "ftp://user:secret@mirror.example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseBaseURL(tt.raw)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidBaseURL)
				assert.Nil(t, got)
				assert.NotContains(t, err.Error(), "secret", "errors must not expose a password")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

// TestClient_endpoint checks that endpoints keep the base path and fall back to the default base URL.
func TestClient_endpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		client  *Client
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:   "no config uses the default",
			client: &Client{},
			path:   serversPath,
			want:   "https://www.speedtest.net/api/js/servers",
		},
		{
			name:   "empty base URL uses the default",
			client: &Client{settings: settings{}},
			path:   userConfigPath,
			want:   "https://www.speedtest.net/speedtest-config.php",
		},
		{
			name:   "path prefix is kept",
			client: &Client{settings: settings{baseURL: "https://mirror.example/speedtest"}},
			path:   serversAdvancedPath,
			want:   "https://mirror.example/speedtest/api/ios-config.php",
		},
		{
			name:   "trailing slash is not doubled",
			client: &Client{settings: settings{baseURL: "https://mirror.example/speedtest/"}},
			path:   serversAlternativePath,
			want:   "https://mirror.example/speedtest/speedtest-servers-static.php",
		},
		{
			name:    "invalid base URL",
			client:  &Client{settings: settings{baseURL: "ftp://mirror.example"}},
			path:    serversPath,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.client.endpoint(tt.path)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidBaseURL)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

// TestNew_BaseURL checks that clients default to DefaultBaseURL and keep a configured base URL.
func TestNew_BaseURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultBaseURL, newTestClient(t).settings.baseURL)
	assert.Equal(
		t,
		"http://127.0.0.1:8080/mirror",
		newBaseURLClient(t, "http://127.0.0.1:8080/mirror").settings.baseURL,
	)
}

// TestFetchUserInfoContext_BaseURL checks that user info is fetched under the configured base path.
func TestFetchUserInfoContext_BaseURL(t *testing.T) {
	t.Parallel()

	api := newMirrorAPI(t)

	user, err := newBaseURLClient(t, api.URL()).FetchUserInfoContext(context.Background())
	require.NoError(t, err)

	want := testserver.DefaultUser()
	assert.Equal(
		t,
		&User{IP: want.IP, Lat: want.Lat, Lon: want.Lon, Isp: want.ISP, Country: want.Country},
		user,
	)
	assert.Equal(
		t,
		mirrorPrefix+testserver.PathUserConfig,
		requestFor(t, api, testserver.PathUserConfig).Path,
	)
}

// TestFetchServerListContext_BaseURL checks that the JSON server list is fetched under the base path with the
// search keyword, and that the latency probes reach the listed servers.
func TestFetchServerListContext_BaseURL(t *testing.T) {
	t.Parallel()

	api := newMirrorAPI(t)
	client := newTestClient(t, WithBaseURL(api.URL()), WithSearch("fiber"))

	servers, err := client.FetchServerListContext(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, len(testserver.DefaultServers()))

	assertProbed(t, api, servers)

	list := requestFor(t, api, testserver.PathServers)
	assert.Equal(t, mirrorPrefix+testserver.PathServers, list.Path)
	assert.Equal(t, "fiber", list.Query.Get("search"))
	assert.False(t, api.Requested(testserver.PathServersStatic))
}

// TestFetchServerListContext_BaseURLFallback checks that an empty JSON response falls back to the XML list
// under the same base path.
func TestFetchServerListContext_BaseURLFallback(t *testing.T) {
	t.Parallel()

	api := newMirrorAPI(t)
	api.SetResponse(testserver.PathServers, testserver.Response{})

	servers, err := newBaseURLClient(t, api.URL()).FetchServerListContext(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, len(testserver.DefaultServers()))

	static := requestFor(t, api, testserver.PathServersStatic)
	assert.Equal(t, mirrorPrefix+testserver.PathServersStatic, static.Path)
}

// TestFetchServerByIDContext_BaseURL checks that a server lookup is sent under the base path with the server ID.
func TestFetchServerByIDContext_BaseURL(t *testing.T) {
	t.Parallel()

	api := newMirrorAPI(t)

	server, err := newBaseURLClient(
		t,
		api.URL(),
	).FetchServerByIDContext(context.Background(), "1001")
	require.NoError(t, err)

	assert.Equal(t, "1001", server.ID)

	lookup := requestFor(t, api, testserver.PathServerLookup)
	assert.Equal(t, mirrorPrefix+testserver.PathServerLookup, lookup.Path)
	assert.Equal(t, "1001", lookup.Query.Get("serverid"))
	assert.NotEmpty(t, lookup.Query.Get(cacheBustParam))
}

// TestFetchers_InvalidBaseURL checks that every fetcher reports an invalid base URL instead of sending a request.
func TestFetchers_InvalidBaseURL(t *testing.T) {
	t.Parallel()

	client := &Client{
		settings: settings{baseURL: "ftp://mirror.example"},
		doer:     http.DefaultClient,
	}

	tests := []struct {
		fetch func() error
		name  string
	}{
		{
			name: "user info",
			fetch: func() error {
				_, err := client.FetchUserInfoContext(context.Background())

				return err
			},
		},
		{
			name: "server list",
			fetch: func() error {
				_, err := client.FetchServerListContext(context.Background())

				return err
			},
		},
		{
			name: "server by ID",
			fetch: func() error {
				_, err := client.FetchServerByIDContext(context.Background(), "1")

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, tt.fetch(), ErrInvalidBaseURL)
		})
	}
}

// TestCacheBust checks that cacheBust sets a fresh random value on each call and keeps the URL's other parameters
// and path.
func TestCacheBust(t *testing.T) {
	t.Parallel()

	values := map[string]bool{}

	for range 3 {
		endpoint, err := url.Parse(
			"https://mirror.example/speedtest/api/ios-config.php?serverid=1001",
		)
		require.NoError(t, err)

		cacheBust(endpoint)

		assert.Equal(t, "/speedtest/api/ios-config.php", endpoint.Path)
		assert.Equal(t, "1001", endpoint.Query().Get("serverid"))

		value := endpoint.Query().Get(cacheBustParam)
		require.NotEmpty(t, value)

		values[value] = true
	}

	assert.Len(t, values, 3, "each call should set a different value")
}

// TestFetchers_CacheBust checks that every request for caller-specific data has a unique URL, so a shared cache
// cannot answer it with another caller's details. Requests made twice must carry different values.
func TestFetchers_CacheBust(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fetch    func(client *Client) error
		name     string
		endpoint string
	}{
		{
			name:     "user info",
			endpoint: testserver.PathUserConfig,
			fetch: func(client *Client) error {
				_, err := client.FetchUserInfoContext(context.Background())

				return err
			},
		},
		{
			name:     "server by ID",
			endpoint: testserver.PathServerLookup,
			fetch: func(client *Client) error {
				_, err := client.FetchServerByIDContext(context.Background(), "1001")

				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := newMirrorAPI(t)
			client := newBaseURLClient(t, api.URL())

			require.NoError(t, tt.fetch(client))
			require.NoError(t, tt.fetch(client))

			var values []string

			for _, req := range api.Requests() {
				if req.Endpoint == tt.endpoint {
					values = append(values, req.Query.Get(cacheBustParam))
				}
			}

			require.Len(t, values, 2)
			assert.NotEmpty(t, values[0])
			assert.NotEmpty(t, values[1])
			assert.NotEqual(t, values[0], values[1])
		})
	}
}

package speedtest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mirrorPrefix is the path prefix the fake API servers are mounted under, so tests prove base paths survive.
const mirrorPrefix = "/mirror"

// fakeAPI is an httptest server that serves canned speedtest API responses and records the paths it receives.
type fakeAPI struct {
	server *httptest.Server
	routes map[string]http.HandlerFunc
	paths  []string
	mu     sync.Mutex
}

// newFakeAPI starts a fake API server with the given routes, keyed by request path.
//
// Unknown paths answer 404 and are still recorded, so a test can show which URL the client built.
func newFakeAPI(t *testing.T, routes map[string]http.HandlerFunc) *fakeAPI {
	t.Helper()

	api := &fakeAPI{routes: routes}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		api.paths = append(api.paths, r.URL.Path)
		api.mu.Unlock()

		handler, ok := api.routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)

			return
		}

		handler(w, r)
	}))
	t.Cleanup(api.server.Close)

	return api
}

// baseURL returns the fake server's URL with the mirror prefix, for use as UserConfig.BaseURL.
func (api *fakeAPI) baseURL() string {
	return api.server.URL + mirrorPrefix
}

// requested reports whether the fake server received a request for path.
func (api *fakeAPI) requested(path string) bool {
	api.mu.Lock()
	defer api.mu.Unlock()

	return slices.Contains(api.paths, path)
}

// writeBody returns a handler that answers with a fixed content type and body.
func writeBody(contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = fmt.Fprint(w, body)
	}
}

// serverEntryXML renders one server element whose upload URL points at the fake API server.
func (api *fakeAPI) serverEntryXML(id string) string {
	host := strings.TrimPrefix(api.server.URL, "http://")

	return fmt.Sprintf(
		`<server url="%s/speedtest/upload.php" lat="35.68" lon="139.76" name="Tokyo" country="Japan" `+
			`sponsor="Fake ISP" id="%s" host="%s"/>`,
		api.server.URL,
		id,
		host,
	)
}

// newBaseURLClient returns a client that talks to the given base URL.
func newBaseURLClient(baseURL string) *Speedtest {
	return New(WithUserConfig(&UserConfig{BaseURL: baseURL}))
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

// TestSpeedtest_endpoint checks that endpoints keep the base path and fall back to the default base URL.
func TestSpeedtest_endpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		client  *Speedtest
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:   "no config uses the default",
			client: &Speedtest{},
			path:   serversPath,
			want:   "https://www.speedtest.net/api/js/servers",
		},
		{
			name:   "empty base URL uses the default",
			client: &Speedtest{config: &UserConfig{}},
			path:   userConfigPath,
			want:   "https://www.speedtest.net/speedtest-config.php",
		},
		{
			name:   "path prefix is kept",
			client: &Speedtest{config: &UserConfig{BaseURL: "https://mirror.example/speedtest"}},
			path:   serversAdvancedPath,
			want:   "https://mirror.example/speedtest/api/ios-config.php",
		},
		{
			name:   "trailing slash is not doubled",
			client: &Speedtest{config: &UserConfig{BaseURL: "https://mirror.example/speedtest/"}},
			path:   serversAlternativePath,
			want:   "https://mirror.example/speedtest/speedtest-servers-static.php",
		},
		{
			name:    "invalid base URL",
			client:  &Speedtest{config: &UserConfig{BaseURL: "ftp://mirror.example"}},
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

// TestNewUserConfig_BaseURL checks that clients default to DefaultBaseURL and keep a configured base URL.
func TestNewUserConfig_BaseURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultBaseURL, New().config.BaseURL)
	assert.Equal(t, DefaultBaseURL, newBaseURLClient("").config.BaseURL)
	assert.Equal(
		t,
		"http://127.0.0.1:8080/mirror",
		newBaseURLClient("http://127.0.0.1:8080/mirror").config.BaseURL,
	)
}

// TestFetchUserInfoContext_BaseURL checks that user info is fetched from the configured base URL.
func TestFetchUserInfoContext_BaseURL(t *testing.T) {
	t.Parallel()

	api := newFakeAPI(t, map[string]http.HandlerFunc{
		mirrorPrefix + "/speedtest-config.php": writeBody(
			"application/xml",
			`<settings><client ip="203.0.113.7" lat="35.68" lon="139.76" isp="Fake ISP"/></settings>`,
		),
	})

	user, err := newBaseURLClient(api.baseURL()).FetchUserInfoContext(context.Background())
	require.NoError(t, err)

	assert.Equal(t, &User{IP: "203.0.113.7", Lat: "35.68", Lon: "139.76", Isp: "Fake ISP"}, user)
}

// TestFetchServerListContext_BaseURL checks that the JSON server list and the latency probe use the base URL.
func TestFetchServerListContext_BaseURL(t *testing.T) {
	t.Parallel()

	var api *fakeAPI

	api = newFakeAPI(t, map[string]http.HandlerFunc{
		mirrorPrefix + "/api/js/servers": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "fiber", r.URL.Query().Get("search"))

			host := strings.TrimPrefix(api.server.URL, "http://")
			writeBody("application/json", fmt.Sprintf(
				`[{"url":"%s/speedtest/upload.php","lat":"35.68","lon":"139.76","name":"Tokyo",`+
					`"country":"Japan","sponsor":"Fake ISP","id":"1","host":"%s"}]`,
				api.server.URL, host,
			))(w, r)
		},
		"/speedtest/latency.txt": writeBody("text/plain", "test=test"),
	})

	client := New(WithUserConfig(&UserConfig{BaseURL: api.baseURL(), Keyword: "fiber"}))

	servers, err := client.FetchServerListContext(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, 1)

	assert.Equal(t, "1", servers[0].ID)
	assert.Positive(t, servers[0].Latency, "the latency probe should reach the fake server")
	assert.False(t, api.requested(mirrorPrefix+"/speedtest-servers-static.php"))
}

// TestFetchServerListContext_BaseURLFallback checks that an empty JSON response falls back to the XML list
// under the same base URL.
func TestFetchServerListContext_BaseURLFallback(t *testing.T) {
	t.Parallel()

	var api *fakeAPI

	api = newFakeAPI(t, map[string]http.HandlerFunc{
		mirrorPrefix + "/api/js/servers": func(http.ResponseWriter, *http.Request) {},
		mirrorPrefix + "/speedtest-servers-static.php": func(w http.ResponseWriter, r *http.Request) {
			body := "<settings><servers>" + api.serverEntryXML("2") + "</servers></settings>"
			writeBody("application/xml", body)(w, r)
		},
		"/speedtest/latency.txt": writeBody("text/plain", "test=test"),
	})

	servers, err := newBaseURLClient(api.baseURL()).FetchServerListContext(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, 1)

	assert.Equal(t, "2", servers[0].ID)
	assert.True(t, api.requested(mirrorPrefix+"/speedtest-servers-static.php"))
}

// TestFetchServerByIDContext_BaseURL checks that a server lookup uses the base URL and sends the server ID.
func TestFetchServerByIDContext_BaseURL(t *testing.T) {
	t.Parallel()

	var api *fakeAPI

	api = newFakeAPI(t, map[string]http.HandlerFunc{
		mirrorPrefix + "/api/ios-config.php": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "3", r.URL.Query().Get("serverid"))

			client := `<client ip="203.0.113.7" lat="34.69" lon="135.50" isp="Fake ISP"/>`
			servers := "<servers>" + api.serverEntryXML("3") + "</servers>"
			body := "<settings>" + client + servers + "</settings>"
			writeBody("application/xml", body)(w, r)
		},
	})

	server, err := newBaseURLClient(api.baseURL()).FetchServerByIDContext(context.Background(), "3")
	require.NoError(t, err)

	assert.Equal(t, "3", server.ID)
	assert.Positive(t, server.Distance, "distance should be computed from the client location")
}

// TestFetchers_InvalidBaseURL checks that every fetcher reports an invalid base URL instead of sending a request.
func TestFetchers_InvalidBaseURL(t *testing.T) {
	t.Parallel()

	client := &Speedtest{
		config: &UserConfig{BaseURL: "ftp://mirror.example"},
		doer:   http.DefaultClient,
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

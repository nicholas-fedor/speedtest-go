package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/task"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// newTestClient builds the client the CLI builds from the settings.
//
// Parameters:
//   - t: the test, which fails when the settings are rejected.
//   - cfg: the settings.
//
// Returns:
//   - *speedtest.Client: the client.
func newTestClient(t *testing.T, cfg config.Config) *speedtest.Client {
	t.Helper()

	client, err := setupSpeedtestClient(cfg)
	require.NoError(t, err)

	return client
}

// TestSetupSpeedtestClient_InvalidSetting checks that a setting only the client checks, such as a proxy without a
// scheme, is an invalid setting, so the CLI exits with code 2.
func TestSetupSpeedtestClient_InvalidSetting(t *testing.T) {
	t.Parallel()

	client, err := setupSpeedtestClient(config.Config{Proxy: "127.0.0.1:3128"})

	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.ErrorIs(t, err, speedtest.ErrInvalidOption)
	assert.Nil(t, client)
}

// serverIDs returns the IDs of servers, in order.
func serverIDs(servers speedtest.Servers) []string {
	ids := make([]string, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, server.ID)
	}

	return ids
}

// TestFetchServersByID checks that the servers come back in the order the IDs were given, not the order the
// concurrent lookups finish.
func TestFetchServersByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []int
		want []string
	}{
		{name: "listed order", ids: []int{1002, 1001}, want: []string{"1002", "1001"}},
		{name: "reversed order", ids: []int{1001, 1002}, want: []string{"1001", "1002"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			client := newTestClient(t, config.Config{BaseURL: api.URL()})

			servers, err := fetchServersByID(client, tt.ids)
			require.NoError(t, err)
			assert.Equal(t, tt.want, serverIDs(servers))
		})
	}
}

// TestFetchServersByIDReportsMissingIDs checks that unknown IDs are an error naming each of them, instead of being
// skipped while the run continues with the rest.
func TestFetchServersByIDReportsMissingIDs(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t, config.Config{BaseURL: api.URL()})

	servers, err := fetchServersByID(client, []int{9999, 1001, 8888})
	require.ErrorIs(t, err, speedtest.ErrServerNotFound)
	assert.Contains(t, err.Error(), "--server 9999, 8888")
	assert.Nil(t, servers)
}

// TestFetchServersByIDReportsLookupFailures checks that a failed lookup is reported as itself, not as a missing ID.
func TestFetchServersByIDReportsLookupFailures(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(testserver.PathServerLookup, testserver.Response{Status: http.StatusBadGateway})
	client := newTestClient(t, config.Config{BaseURL: api.URL()})

	_, err := fetchServersByID(client, []int{1001})
	require.ErrorIs(t, err, speedtest.ErrUnexpectedStatus)
	require.NotErrorIs(t, err, speedtest.ErrServerNotFound)
	assert.Contains(t, err.Error(), "server 1001")
}

// TestRetrieveServersPool checks the pool that --multi tests against. Servers named with --custom-url or --server
// are their own pool. Otherwise the pool is the fetched server list.
func TestRetrieveServersPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cfg         func(api *testserver.API) config.Config
		name        string
		wantPool    []string
		wantTargets []string
	}{
		{
			name: "server IDs",
			cfg: func(api *testserver.API) config.Config {
				return config.Config{BaseURL: api.URL(), ServerIDs: []int{1001, 1002}}
			},
			wantPool:    []string{"1001", "1002"},
			wantTargets: []string{"1001", "1002"},
		},
		{
			name: "custom URL",
			cfg: func(api *testserver.API) config.Config {
				return config.Config{BaseURL: api.URL(), CustomURL: api.URL()}
			},
			wantPool:    []string{"Custom"},
			wantTargets: []string{"Custom"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			cfg := tt.cfg(api)

			pool, targets := retrieveServers(
				newTestClient(t, cfg),
				cfg,
				task.NewManager(true, false),
			)
			assert.Equal(t, tt.wantPool, serverIDs(pool))
			assert.Equal(t, tt.wantTargets, serverIDs(targets))
		})
	}

	t.Run("fetched list", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		cfg := config.Config{BaseURL: api.URL()}

		pool, targets := retrieveServers(
			newTestClient(t, cfg),
			cfg,
			task.NewManager(true, false),
		)
		assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(pool))
		assert.Len(t, targets, 1)
	})
}

// TestFilterByCountry checks the country filter: explicit codes, auto from the user's country, auto from the
// city, which takes precedence over the user's country, and the error when no server matches. Both fake servers
// are in JP, so a match keeps the whole list in its order.
func TestFilterByCountry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr   error
		name      string
		city      string
		codes     []string
		wantCodes []string
	}{
		{name: "explicit", codes: []string{"JP"}, wantCodes: []string{"JP"}},
		{name: "auto from user", codes: []string{"AUTO"}, wantCodes: []string{"JP"}},
		{name: "auto from city", codes: []string{"AUTO"}, city: "tokyo", wantCodes: []string{"JP"}},
		{
			name:      "city over user",
			codes:     []string{"AUTO"},
			city:      "london",
			wantCodes: []string{"GB"},
			wantErr:   errNoServersInCountries,
		},
		{
			name:      "no match",
			codes:     []string{"FR", "DE"},
			wantCodes: []string{"FR", "DE"},
			wantErr:   errNoServersInCountries,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			client := newTestClient(t, config.Config{BaseURL: api.URL()})

			servers, err := client.FetchServers()
			require.NoError(t, err)

			filtered, codes, err := filterByCountry(
				client,
				config.Config{CountryCodes: tt.codes, City: tt.city},
				servers,
			)
			assert.Equal(t, tt.wantCodes, codes)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), tt.wantCodes[0])

				return
			}

			require.NoError(t, err)
			assert.Equal(t, serverIDs(servers), serverIDs(filtered))
		})
	}
}

// TestAutoCountryCity checks auto with a --city label: an unknown label is an error rather than a silent switch to
// the user's country, and a label registered without a country falls back to the user's country.
func TestAutoCountryCity(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t, config.Config{BaseURL: api.URL()})

	_, err := autoCountry(client, config.Config{City: "atlantis"})
	require.ErrorIs(t, err, errUnknownCountry)
	require.ErrorIs(t, err, speedtest.ErrNotFoundLocation)
	assert.Contains(t, err.Error(), `--city "atlantis"`)

	speedtest.NewLocation("country-filter-no-cc", 35.6, 139.7)

	country, err := autoCountry(client, config.Config{City: "country-filter-no-cc"})
	require.NoError(t, err)
	assert.Equal(t, "JP", country, "a label without a country uses the user's country")
}

// TestAutoCountryUnknown checks that auto fails clearly when speedtest.net reports no country for the user.
func TestAutoCountryUnknown(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(
		t,
		testserver.WithUser(testserver.User{IP: "203.0.113.7", Lat: "35.6", Lon: "139.7"}),
	)
	client := newTestClient(t, config.Config{BaseURL: api.URL()})

	_, err := autoCountry(client, config.Config{})
	require.ErrorIs(t, err, errUnknownCountry)
}

// TestRetrieveServersCountryFilter checks that the speed test picks its target from the filtered servers.
func TestRetrieveServersCountryFilter(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	cfg := config.Config{BaseURL: api.URL(), CountryCodes: []string{"JP"}}

	pool, targets := retrieveServers(newTestClient(t, cfg), cfg, task.NewManager(true, false))
	assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(pool))
	assert.Len(t, targets, 1)
}

// TestSetupSpeedtestClientLocation checks that the location and search settings reach the server list request, so
// the speed test, not only list, honors them.
func TestSetupSpeedtestClientLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cfg     func(baseURL string) config.Config
		name    string
		wantLat string
		wantLon string
	}{
		{
			name: "city",
			cfg: func(baseURL string) config.Config {
				return config.Config{BaseURL: baseURL, City: "tokyo", Search: "fiber"}
			},
			wantLat: "35.680938",
			wantLon: "139.7674114",
		},
		{
			name: "coordinates",
			cfg: func(baseURL string) config.Config {
				return config.Config{BaseURL: baseURL, Location: "60,-110", Search: "fiber"}
			},
			wantLat: "60",
			wantLon: "-110",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)

			_, err := newTestClient(t, tt.cfg(api.URL())).FetchServers()
			require.NoError(t, err)

			var found bool

			for _, req := range api.Requests() {
				if req.Endpoint != testserver.PathServers {
					continue
				}

				found = true

				assert.Equal(t, tt.wantLat, req.Query.Get("lat"))
				assert.Equal(t, tt.wantLon, req.Query.Get("lon"))
				assert.Equal(t, "fiber", req.Query.Get("search"))
			}

			assert.True(t, found, "the server list must be requested")
		})
	}
}

// TestReportedUser checks that --mask-isp masks the reported user and its ISP line, while the client's stored user
// keeps its real values for anything else that reads it.
func TestReportedUser(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, config.Config{})
	client.SetUser(
		&speedtest.User{IP: "203.0.113.7", Lat: "35.6812", Lon: "139.7671", Isp: "Example ISP"},
	)

	plain := reportedUser(client.GetUser(), config.Config{})
	assert.Same(t, client.GetUser(), plain, "without --mask-isp the user is reported as is")

	masked := reportedUser(client.GetUser(), config.Config{MaskISP: true})
	assert.Equal(t, "***.*.***.* (Example ISP) [**.****, ***.****] ", masked.String())
	assert.Equal(
		t,
		"203.0.113.7",
		client.GetUser().IP,
		"the client's user must keep its real address",
	)

	assert.Nil(t, reportedUser(nil, config.Config{MaskISP: true}))
}

// TestServerLine checks that --mask-isp masks the distance in the Test Server line.
func TestServerLine(t *testing.T) {
	t.Parallel()

	server := &speedtest.Server{
		ID:       "6691",
		Distance: 9.03,
		Name:     "Shizuoka",
		Country:  "Japan",
		Sponsor:  "sudosan",
	}

	assert.Equal(
		t,
		"[6691] 9.03km Shizuoka (Japan) by sudosan",
		serverLine(server, config.Config{}),
	)
	assert.Equal(
		t,
		"[6691] *.**km Shizuoka (Japan) by sudosan",
		serverLine(server, config.Config{MaskISP: true}),
	)
}

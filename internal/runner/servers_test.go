// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

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

			servers, err := fetchServersByID(
				t.Context(),
				newClient(t, testserver.NewAPI(t)),
				tt.ids,
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, serverIDs(servers))
		})
	}
}

// TestFetchServersByIDReportsMissingIDs checks that unknown IDs are an error naming each of them, instead of being
// skipped while the run continues with the rest.
func TestFetchServersByIDReportsMissingIDs(t *testing.T) {
	t.Parallel()

	servers, err := fetchServersByID(
		t.Context(),
		newClient(t, testserver.NewAPI(t)),
		[]int{9999, 1001, 8888},
	)
	require.ErrorIs(t, err, speedtest.ErrServerNotFound)
	assert.Contains(t, err.Error(), "--server 9999, 8888")
	assert.Nil(t, servers)
}

// TestFetchServersByIDReportsLookupFailures checks that a failed lookup is reported as itself, not as a missing ID.
func TestFetchServersByIDReportsLookupFailures(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(testserver.PathServerLookup, testserver.Response{Status: http.StatusBadGateway})

	_, err := fetchServersByID(t.Context(), newClient(t, api), []int{1001})
	require.ErrorIs(t, err, speedtest.ErrUnexpectedStatus)
	require.NotErrorIs(t, err, speedtest.ErrServerNotFound)
	assert.Contains(t, err.Error(), "server 1001")
}

// TestRunner_lookup checks the servers each setting selects, and the pool that a multi-server test spreads across:
// servers named by ID or URL are their own pool, and the server list is the pool of its lowest-latency server.
func TestRunner_lookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		settings    func(api *testserver.API) Settings
		name        string
		wantPool    []string
		wantTargets []string
		wantFound   ServersFound
	}{
		{
			name:        "server IDs",
			settings:    func(*testserver.API) Settings { return Settings{ServerIDs: []int{1001, 1002}} },
			wantPool:    []string{"1001", "1002"},
			wantTargets: []string{"1001", "1002"},
			wantFound:   ServersFound{Codes: nil, Count: 2, Source: SourceSpecified},
		},
		{
			name:        "custom URL",
			settings:    func(api *testserver.API) Settings { return Settings{CustomURL: api.URL()} },
			wantPool:    []string{"Custom"},
			wantTargets: []string{"Custom"},
			wantFound:   ServersFound{Codes: nil, Count: 1, Source: SourceCustom},
		},
		{
			name:        "country filter",
			settings:    func(*testserver.API) Settings { return Settings{CountryCodes: []string{CountryAuto}} },
			wantPool:    []string{"1002", "1001"},
			wantTargets: nil,
			wantFound:   ServersFound{Codes: []string{"JP"}, Count: 2, Source: SourceListed},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			events := &recorder{events: nil, mu: sync.Mutex{}}

			found, err := New(newClient(t, api), events, tt.settings(api)).lookup(t.Context())
			require.NoError(t, err)

			assert.ElementsMatch(t, tt.wantPool, serverIDs(found.pool))

			if tt.wantTargets != nil {
				assert.Equal(t, tt.wantTargets, serverIDs(found.targets))
			} else {
				assert.Len(
					t,
					found.targets,
					1,
					"the server list's target is its lowest-latency server",
				)
			}

			assert.Contains(t, events.events, tt.wantFound)
		})
	}
}

// TestListServers checks the list command's country filter: explicit codes, auto from the user's country, auto from
// the city, which takes precedence over the user's country, and the errors when no server matches or no country is
// known. Both fake servers are in JP, so a match keeps the whole list in its order.
func TestListServers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		options   []testserver.Option
		wantErr   error
		name      string
		city      string
		codes     []string
		wantCodes []string
		wantIDs   []string
	}{
		{name: "no filter", wantIDs: []string{"1002", "1001"}},
		{
			name:      "explicit",
			codes:     []string{"JP"},
			wantCodes: []string{"JP"},
			wantIDs:   []string{"1002", "1001"},
		},
		{
			name:      "auto from user",
			codes:     []string{CountryAuto},
			wantCodes: []string{"JP"},
			wantIDs:   []string{"1002", "1001"},
		},
		{
			name:      "auto from city",
			codes:     []string{CountryAuto},
			city:      "tokyo",
			wantCodes: []string{"JP"},
			wantIDs:   []string{"1002", "1001"},
		},
		{
			name:      "city over user",
			codes:     []string{CountryAuto},
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
		{
			name:    "unknown city",
			codes:   []string{CountryAuto},
			city:    "atlantis",
			wantErr: speedtest.ErrNotFoundLocation,
		},
		{
			name: "user without a country",
			options: []testserver.Option{
				testserver.WithUser(testserver.User{IP: "203.0.113.7", Lat: "35.6", Lon: "139.7"}),
			},
			codes:   []string{CountryAuto},
			wantErr: errUnknownCountry,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t, tt.options...)

			servers, codes, err := ListServers(
				t.Context(),
				newClient(t, api),
				Settings{CountryCodes: tt.codes, City: tt.city},
			)
			assert.Equal(t, tt.wantCodes, codes)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantIDs, serverIDs(servers))
		})
	}

	t.Run("a city label never falls back to the user", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)

		_, _, err := ListServers(
			t.Context(),
			newClient(t, api),
			Settings{CountryCodes: []string{CountryAuto}, City: "atlantis"},
		)
		require.ErrorIs(t, err, errUnknownCountry)
		assert.Contains(t, err.Error(), `--city "atlantis"`)
		assert.False(t, api.Requested(testserver.PathUserConfig))
	})
}

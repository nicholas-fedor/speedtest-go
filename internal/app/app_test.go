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
			client := setupSpeedtestClient(config.Config{BaseURL: api.URL()})

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
	client := setupSpeedtestClient(config.Config{BaseURL: api.URL()})

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
	client := setupSpeedtestClient(config.Config{BaseURL: api.URL()})

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
				setupSpeedtestClient(cfg),
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
			setupSpeedtestClient(cfg),
			cfg,
			task.NewManager(true, false),
		)
		assert.ElementsMatch(t, []string{"1001", "1002"}, serverIDs(pool))
		assert.Len(t, targets, 1)
	})
}

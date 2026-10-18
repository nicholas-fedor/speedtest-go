// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/report"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
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

			_, err := newTestClient(t, tt.cfg(api.URL())).FetchServers(t.Context())
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

// TestRunSpeedtest_JSON checks a --json run against the fake API: the output is exactly one JSON document, with the
// tested server, and no progress.
func TestRunSpeedtest_JSON(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var out bytes.Buffer

	err := runSpeedtest(
		t.Context(),
		config.Config{BaseURL: api.URL(), JSONOutput: true, NoDownload: true, NoUpload: true},
		&out,
	)
	require.NoError(t, err)

	decoder := json.NewDecoder(&out)

	var document map[string]any
	require.NoError(t, decoder.Decode(&document))
	assert.False(t, decoder.More(), "the output must be one JSON document")

	servers, ok := document["servers"].([]any)
	require.True(t, ok)
	assert.Len(t, servers, 1)
}

// TestRunSpeedtest_JSONL checks that a --jsonl run writes one line per tested server.
func TestRunSpeedtest_JSONL(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var out bytes.Buffer

	err := runSpeedtest(
		t.Context(),
		config.Config{
			BaseURL:     api.URL(),
			JSONLOutput: true,
			ServerIDs:   []int{1001, 1002},
			NoDownload:  true,
			NoUpload:    true,
		},
		&out,
	)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	require.Len(t, lines, 2)

	for _, line := range lines {
		assert.True(t, json.Valid([]byte(line)), line)
	}
}

// TestRunSpeedtest_Interrupted checks that an interrupted run returns the context's error and writes no report.
func TestRunSpeedtest_Interrupted(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out bytes.Buffer

	err := runSpeedtest(ctx, config.Config{BaseURL: api.URL(), JSONOutput: true}, &out)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, out.String())
}

// TestRunSpeedtest_Unix checks that --unix output shows the banner and the result lines without spinners.
func TestRunSpeedtest_Unix(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var out bytes.Buffer

	err := runSpeedtest(
		t.Context(),
		config.Config{BaseURL: api.URL(), UnixOutput: true, NoDownload: true, NoUpload: true},
		&out,
	)
	require.NoError(t, err)

	assert.Contains(t, out.String(), "speedtest-go")
	assert.Contains(t, out.String(), "\nFound 2 Public Servers\n")
	assert.Contains(t, out.String(), "\nPacket Loss: N/A\n")
}

// TestRunList checks that the list command writes one line per server, and the count of servers found as a plain
// status line apart from the list, with no log timestamp.
func TestRunList(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var out, status bytes.Buffer

	require.NoError(t, runList(t.Context(), config.Config{BaseURL: api.URL()}, &out, &status))
	assert.Len(t, strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"), 2)
	assert.Equal(t, "Found 2 Public Servers\n", status.String())
}

// TestRunListInterrupted checks that the list command returns the context's error when it ends the server lookup.
func TestRunListInterrupted(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := runList(ctx, config.Config{BaseURL: api.URL()}, &bytes.Buffer{}, &bytes.Buffer{})
	require.ErrorIs(t, err, context.Canceled)
}

// TestNewReporter checks that the settings select the reporter: nothing besides the JSON reports, and lines for
// --unix, --debug, and output that is not a terminal, such as a file or a pipe.
func TestNewReporter(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	assert.IsType(t, report.Discard{}, newReporter(config.Config{JSONOutput: true}, &out))
	assert.IsType(t, report.Discard{}, newReporter(config.Config{JSONLOutput: true}, &out))
	assert.IsType(t, &report.Lines{}, newReporter(config.Config{UnixOutput: true}, &out))
	assert.IsType(t, &report.Lines{}, newReporter(config.Config{Debug: true}, &out))
	assert.IsType(t, &report.Lines{}, newReporter(config.Config{}, &out))

	file, err := os.Create(filepath.Join(t.TempDir(), "output"))
	require.NoError(t, err)

	t.Cleanup(func() { _ = file.Close() })

	assert.False(t, isTerminal(file), "a file is not a terminal")
	assert.IsType(t, &report.Lines{}, newReporter(config.Config{}, file))
}

// TestRunSettings checks that the settings reach the run, and that the country code for the user's own country
// means the same in both packages.
func TestRunSettings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, config.CountryAuto, runner.CountryAuto)

	settings := runSettings(config.Config{
		CustomURL:    "http://example.com",
		City:         "tokyo",
		ServerIDs:    []int{1},
		CountryCodes: []string{config.CountryAuto},
		Multi:        true,
		NoDownload:   true,
		NoUpload:     true,
	})

	assert.Equal(t, runner.Settings{
		CustomURL:    "http://example.com",
		City:         "tokyo",
		ServerIDs:    []int{1},
		CountryCodes: []string{runner.CountryAuto},
		Multi:        true,
		NoDownload:   true,
		NoUpload:     true,
	}, settings)
}

// failingWriter fails every write.
type failingWriter struct{}

// Write fails.
func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("disk full")
}

// TestWriteReports_WriteError checks that a report that cannot be written is an error, not a silent success.
func TestWriteReports_WriteError(t *testing.T) {
	t.Parallel()

	results := []speedtest.Result{{Server: &speedtest.Server{ID: "1001"}}}

	require.ErrorContains(
		t,
		writeReports(failingWriter{}, config.Config{JSONOutput: true}, nil, results),
		"disk full",
	)
	require.ErrorContains(
		t,
		writeReports(failingWriter{}, config.Config{JSONLOutput: true}, nil, results),
		"disk full",
	)
	require.NoError(t, writeReports(failingWriter{}, config.Config{}, nil, results))
}

// TestSilenceStandardLogger checks that the standard logger is silenced only until the returned function runs, so
// messages after a run still show.
//
//nolint:paralleltest // The test changes the standard logger, which every package shares.
func TestSilenceStandardLogger(t *testing.T) {
	var shown bytes.Buffer

	previous := log.Writer()

	log.SetOutput(&shown)

	t.Cleanup(func() { log.SetOutput(previous) })

	restore := silenceStandardLogger()

	log.Print("during the run")

	restore()
	log.Print("after the run")

	assert.NotContains(t, shown.String(), "during the run")
	assert.Contains(t, shown.String(), "after the run")
}

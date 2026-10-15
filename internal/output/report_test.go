// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
)

// update rewrites the golden files from the current output instead of comparing against them.
var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// reportTimestamp is the fixed time of the reports in the golden files.
var reportTimestamp = time.Date(
	2026,
	10,
	13,
	8,
	30,
	15,
	123_000_000,
	time.UTC,
)

// reportUser returns the user in the golden reports.
func reportUser() *speedtest.User {
	return &speedtest.User{
		IP:      "203.0.113.7",
		Lat:     "35.6812",
		Lon:     "139.7671",
		Isp:     "Example ISP",
		Country: "JP",
	}
}

// reportResults returns the results in the golden reports: one server with every measurement, and one whose
// download and upload rates were not available and that has no other measurement.
func reportResults() []speedtest.Result {
	return []speedtest.Result{
		{
			Server: &speedtest.Server{
				URL:      "http://speed.example/speedtest/upload.php",
				Lat:      "35.6895",
				Lon:      "139.6917",
				Name:     "Tokyo",
				Country:  "Japan",
				CC:       "JP",
				Sponsor:  "Example Networks",
				ID:       "1001",
				Host:     "speed.example:8080",
				Distance: 6.95,
				Latency:  0,
			},
			Ping: &speedtest.PingStats{
				Latencies: nil,
				Latency:   4500 * time.Microsecond,
				Jitter:    300 * time.Microsecond,
				Min:       4 * time.Millisecond,
				Max:       5 * time.Millisecond,
				Duration:  2 * time.Second,
			},
			Download:   &speedtest.Transfer{Rate: 12_500_000, Duration: 10 * time.Second},
			Upload:     &speedtest.Transfer{Rate: 2_500_000, Duration: 10 * time.Second},
			PacketLoss: &transport.PLoss{Sent: 300, Dup: 1, Max: 301},
		},
		{
			Server: &speedtest.Server{
				URL:      "http://speed2.example/speedtest/upload.php",
				Lat:      "34.6937",
				Lon:      "135.5023",
				Name:     "Osaka",
				Country:  "Japan",
				CC:       "JP",
				Sponsor:  "Example Networks",
				ID:       "1002",
				Host:     "speed2.example:8080",
				Distance: 396.5,
				Latency:  0,
			},
			Ping:       nil,
			Download:   &speedtest.Transfer{Rate: -1},
			Upload:     &speedtest.Transfer{Rate: -1},
			PacketLoss: nil,
		},
	}
}

// renderReport renders a JSON or JSONL report the way the CLI prints it, masked for --mask-isp when masked is set.
func renderReport(t *testing.T, jsonl, masked bool, user *speedtest.User) []byte {
	t.Helper()

	jsonReport, jsonlReport := JSONReport, JSONLReport
	if masked {
		jsonReport, jsonlReport = MaskedJSONReport, MaskedJSONLReport
	}

	if !jsonl {
		data, err := jsonReport(reportTimestamp, user, reportResults())
		require.NoError(t, err)

		return data
	}

	var lines bytes.Buffer

	for _, result := range reportResults() {
		data, err := jsonlReport(reportTimestamp, user, result)
		require.NoError(t, err)

		lines.Write(data)
		lines.WriteByte('\n')
	}

	return lines.Bytes()
}

// TestReportGolden checks the JSON and JSONL reports byte for byte against the golden files, so a change to the
// report format is always a deliberate one. Run with -update to rewrite them.
func TestReportGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		user   *speedtest.User
		name   string
		golden string
		jsonl  bool
		masked bool
	}{
		{name: "json", golden: "report.json", user: reportUser()},
		{name: "jsonl", golden: "report.jsonl", user: reportUser(), jsonl: true},
		{name: "json masked", golden: "report_masked.json", user: reportUser(), masked: true},
		{
			name:   "jsonl masked",
			golden: "report_masked.jsonl",
			user:   reportUser(),
			jsonl:  true,
			masked: true,
		},
		{name: "json without user", golden: "report_no_user.json", user: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderReport(t, tt.jsonl, tt.masked, tt.user)
			path := filepath.Join("testdata", tt.golden)

			if *update {
				require.NoError(t, os.WriteFile(path, got, 0o600))
			}

			want, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(got))
		})
	}
}

// withoutTimestamp decodes a JSON report and drops its timestamp, so reports made at different times compare equal.
func withoutTimestamp(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var decoded map[string]any

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Contains(t, decoded, "timestamp")
	delete(decoded, "timestamp")

	return decoded
}

// TestReportMatchesLibrary checks that the CLI's reports have the same content as the library's JSON and JSONL,
// which they replace in the CLI.
func TestReportMatchesLibrary(t *testing.T) {
	t.Parallel()

	library, err := speedtest.JSON(reportUser(), reportResults())
	require.NoError(t, err)

	assert.Equal(
		t,
		withoutTimestamp(t, library),
		withoutTimestamp(t, renderReport(t, false, false, reportUser())),
	)

	result := reportResults()[0]

	libraryLine, err := speedtest.JSONL(reportUser(), result)
	require.NoError(t, err)

	line, err := JSONLReport(reportTimestamp, reportUser(), result)
	require.NoError(t, err)

	assert.Equal(t, withoutTimestamp(t, libraryLine), withoutTimestamp(t, line))
}

// TestMaskedReport checks that a masked report differs from the unmasked one only in the masked user details and the
// missing server distances, and that the given user is left unchanged.
func TestMaskedReport(t *testing.T) {
	t.Parallel()

	user := reportUser()

	masked := withoutTimestamp(t, renderReport(t, false, true, user))
	plain := withoutTimestamp(t, renderReport(t, false, false, reportUser()))

	assert.Equal(t, "203.0.113.7", user.IP, "the given user must not change")

	maskedUser, ok := masked["userInfo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "***.*.***.*", maskedUser["ip"])

	maskedServers, ok := masked["servers"].([]any)
	require.True(t, ok)

	plainServers, ok := plain["servers"].([]any)
	require.True(t, ok)
	require.Len(t, maskedServers, len(plainServers))

	for index := range plainServers {
		want, ok := plainServers[index].(map[string]any)
		require.True(t, ok)

		delete(want, "distance")
		assert.Equal(t, want, maskedServers[index], "only the distance may differ")
	}
}

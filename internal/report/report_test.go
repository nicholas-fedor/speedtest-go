// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// testUser is the caller in the reported events.
var testUser = &speedtest.User{
	IP:  "203.0.113.7",
	Lat: "35.6812",
	Lon: "139.7671",
	Isp: "Example ISP",
}

// testServer is the server in the reported events.
var testServer = &speedtest.Server{
	ID:       "6691",
	Distance: 9.03,
	Name:     "Shizuoka",
	Country:  "Japan",
	Sponsor:  "sudosan",
}

// runEvents returns the events of a run that tests one server, with live progress.
func runEvents() []runner.Event {
	loaded := speedtest.PingStats{
		Latency: 22 * time.Millisecond,
		Jitter:  2 * time.Millisecond,
		Min:     17 * time.Millisecond,
		Max:     24 * time.Millisecond,
	}

	return []runner.Event{
		runner.LookupStarted{},
		runner.UserFound{User: testUser},
		runner.ServersFound{Codes: []string{"JP", "GB"}, Count: 20, Source: runner.SourceListed},
		runner.ServerStarted{Server: testServer},
		runner.PingStarted{},
		runner.LatencyMeasured{Latency: 4 * time.Millisecond},
		runner.PingFinished{Stats: speedtest.PingStats{
			Latency: 4452963 * time.Nanosecond,
			Jitter:  41271 * time.Nanosecond,
			Min:     4395179 * time.Nanosecond,
			Max:     4517576 * time.Nanosecond,
		}},
		runner.PacketLossStarted{},
		runner.TransferStarted{Direction: runner.Download},
		runner.TransferProgress{Direction: runner.Download, Rate: 12_500_000, Latency: 0},
		runner.TransferFinished{
			Latency:   loaded,
			Transfer:  speedtest.Transfer{Rate: 14_440_000, Bytes: 135_750_000},
			Direction: runner.Download,
		},
		runner.TransferStarted{Direction: runner.Upload},
		runner.TransferProgress{
			Direction: runner.Upload,
			Rate:      500_000,
			Latency:   21 * time.Millisecond,
		},
		runner.TransferFinished{
			Latency:   loaded,
			Transfer:  speedtest.Transfer{Rate: 502_500, Bytes: 6_850_000, Sent: 6_850_000},
			Direction: runner.Upload,
		},
		runner.PacketLossFinished{Stats: speedtest.PacketLossStats{Sent: 217, Dup: 0, Max: 237}},
		runner.ServerFinished{Result: speedtest.Result{Server: testServer}},
		runner.RunFinished{Err: nil},
	}
}

// resultLines are the lines a run of [runEvents] shows, in the format of earlier releases.
var resultLines = []string{
	"ISP: 203.0.113.7 (Example ISP) [35.6812, 139.7671] ",
	"Found 20 Public Servers in JP, GB",
	"",
	"Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan",
	"Latency: 4.452963ms Jitter: 41.271µs Min: 4.395179ms Max: 4.517576ms",
	"Packet Loss Analyzer: Running in background (<= 30 Secs)",
	"Download: 115.52 Mbps (Used: 135.75MB) (Latency: 22ms Jitter: 2ms Min: 17ms Max: 24ms)",
	"Upload: 4.02 Mbps (Used: 6.85MB Confirmed: 100.0%) (Latency: 22ms Jitter: 2ms Min: 17ms Max: 24ms)",
	"Packet Loss: 8.82% (Sent: 217/Dup: 0/Max: 237)",
}

// report sends events to a reporter.
func report(reporter runner.Reporter, events []runner.Event) {
	for _, event := range events {
		reporter.Report(event)
	}
}

// TestLines checks that the line reporter writes each result line, and nothing for progress events.
func TestLines(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	report(
		NewLines(&out, Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false}),
		runEvents(),
	)

	assert.Equal(t, strings.Join(resultLines, "\n")+"\n", out.String())
}

// TestLines_MaskISP checks that --mask-isp masks the user's address and coordinates and the server's distance.
func TestLines_MaskISP(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	report(
		NewLines(&out, Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: true}),
		[]runner.Event{
			runner.UserFound{User: testUser},
			runner.ServerStarted{Server: testServer},
		},
	)

	assert.Equal(
		t,
		"ISP: ***.*.***.* (Example ISP) [**.****, ***.****] \n\nTest Server: [6691] *.**km Shizuoka (Japan) by sudosan\n",
		out.String(),
	)
	assert.Equal(t, "203.0.113.7", testUser.IP, "the reported user must keep its real address")
}

// TestServersLine checks the line for each source of servers.
func TestServersLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		found runner.ServersFound
		want  string
	}{
		{
			name:  "listed",
			found: runner.ServersFound{Codes: nil, Count: 20, Source: runner.SourceListed},
			want:  "Found 20 Public Servers",
		},
		{
			name: "listed in countries",
			found: runner.ServersFound{
				Codes:  []string{"GB"},
				Count:  3,
				Source: runner.SourceListed,
			},
			want: "Found 3 Public Servers in GB",
		},
		{
			name:  "specified",
			found: runner.ServersFound{Codes: nil, Count: 2, Source: runner.SourceSpecified},
			want:  "Found 2 Specified Public Server(s)",
		},
		{
			name:  "custom",
			found: runner.ServersFound{Codes: nil, Count: 1, Source: runner.SourceCustom},
			want:  "Skip: Using Custom Server",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, serversLine(tt.found))
		})
	}
}

// TestProgressLine checks the live transfer line, before and after the first loaded round trip.
func TestProgressLine(t *testing.T) {
	t.Parallel()

	options := Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false}

	assert.Equal(t, "Download: 100.00 Mbps (Latency: --)", progressLine(runner.TransferProgress{
		Direction: runner.Download,
		Rate:      12_500_000,
		Latency:   0,
	}, options))
	assert.Equal(t, "Upload: 4.00 Mbps (Latency: 21ms)", progressLine(runner.TransferProgress{
		Direction: runner.Upload,
		Rate:      500_000,
		Latency:   21 * time.Millisecond,
	}, options))
}

// messages returns the message and state of each spinner of the current group.
func messages(s *Spinner) []string {
	spinners := s.manager.GetSpinners()
	result := make([]string, 0, len(spinners))

	for _, spinner := range spinners {
		state := "running"

		switch {
		case spinner.IsError():
			state = "error"
		case spinner.IsComplete():
			state = "done"
		default:
		}

		result = append(result, state+": "+spinner.GetMessage())
	}

	return result
}

// TestSpinner checks that each lookup and test gets a spinner that shows its progress and turns into its result
// line, and that each server starts a new group.
func TestSpinner(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner(io.Discard, Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false})
	events := runEvents()

	report(spinner, events[:2])
	assert.Equal(
		t,
		[]string{"done: " + resultLines[0], "running: Retrieving Servers"},
		messages(spinner),
	)

	report(spinner, events[2:3])
	assert.Equal(
		t,
		[]string{"done: " + resultLines[0], "done: " + resultLines[1]},
		messages(spinner),
	)

	report(spinner, events[3:6])
	assert.Equal(t, []string{"done: " + resultLines[3], "running: Latency: 4ms"}, messages(spinner))

	report(spinner, events[6:10])
	assert.Equal(t, []string{
		"done: " + resultLines[3],
		"done: " + resultLines[4],
		"done: " + resultLines[5],
		"running: Download: 100.00 Mbps (Latency: --)",
	}, messages(spinner))

	report(spinner, events[10:])
	assert.Equal(t, []string{
		"done: " + resultLines[3],
		"done: " + resultLines[4],
		"done: " + resultLines[5],
		"done: " + resultLines[6],
		"done: " + resultLines[7],
		"done: " + resultLines[8],
	}, messages(spinner))
}

// TestSpinner_Failed checks that a failed run marks its unfinished spinners as failed.
func TestSpinner_Failed(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner(io.Discard, Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false})

	report(spinner, []runner.Event{
		runner.LookupStarted{},
		runner.UserFound{User: testUser},
		runner.RunFinished{Err: errors.New("connection refused")},
	})

	assert.Equal(
		t,
		[]string{"done: " + resultLines[0], "error: Retrieving Servers"},
		messages(spinner),
	)
}

// TestDiscard checks that the discard reporter accepts every event.
func TestDiscard(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() { report(Discard{}, runEvents()) })
}

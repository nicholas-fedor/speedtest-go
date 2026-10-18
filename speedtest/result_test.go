// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_outputTime_MarshalJSON checks the quoted, millisecond-precision timestamp of the JSON reports.
func Test_outputTime_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tr      outputTime
		want    []byte
		wantErr bool
	}{
		{
			name:    "marshal output time",
			tr:      outputTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)),
			want:    []byte(`"2023-01-01 12:00:00.000"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.tr.MarshalJSON()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestJSON checks that the report holds the given user and results.
func TestJSON(t *testing.T) {
	t.Parallel()

	got, err := JSON(&User{IP: "203.0.113.7"}, []Result{{Server: &Server{ID: "1001"}}})
	require.NoError(t, err)

	var report struct {
		Timestamp string    `json:"timestamp"`
		UserInfo  *User     `json:"userInfo"`
		Servers   []*Server `json:"servers"`
	}

	require.NoError(t, json.Unmarshal(got, &report))
	assert.NotEmpty(t, report.Timestamp)
	assert.Equal(t, "203.0.113.7", report.UserInfo.IP)
	require.Len(t, report.Servers, 1)
	assert.Equal(t, "1001", report.Servers[0].ID)
}

// TestJSONL checks that the line holds the given user and result, and that a missing user is null.
func TestJSONL(t *testing.T) {
	t.Parallel()

	got, err := JSONL(&User{IP: "203.0.113.7"}, Result{Server: &Server{ID: "1001"}})
	require.NoError(t, err)
	assert.NotContains(t, string(got), "\n")
	assert.Contains(t, string(got), `"userInfo":{"ip":"203.0.113.7"`)
	assert.Contains(
		t,
		string(got),
		`"server":{"url":"","lat":"","lon":"","name":"","country":"","cc":"","sponsor":"","id":"1001"`,
	)

	got, err = JSONL(nil, Result{Server: &Server{}})
	require.NoError(t, err)
	assert.Contains(t, string(got), `"userInfo":null`)
}

// TestResult_Report checks the report of a result with every measurement, and of one without any, whose durations
// are null and whose measured values are zero.
func TestResult_Report(t *testing.T) {
	t.Parallel()

	server := &Server{ID: "1001", Name: "Tokyo", Host: "speed.example:8080", Distance: 6.95}
	duration := func(value time.Duration) *time.Duration { return &value }

	full := Result{
		Server: server,
		Ping: &PingStats{
			Latencies: nil,
			Latency:   4 * time.Millisecond,
			Jitter:    time.Millisecond,
			Min:       3 * time.Millisecond,
			Max:       6 * time.Millisecond,
			Duration:  2 * time.Second,
		},
		Download:   &Transfer{Rate: 12_500_000, Duration: 10 * time.Second},
		Upload:     &Transfer{Rate: 2_500_000, Duration: 10 * time.Second},
		PacketLoss: &PacketLossStats{Sent: 300, Dup: 1, Max: 301},
	}

	report := full.Report()

	require.NotNil(t, report.Distance)
	assert.InDelta(t, 6.95, *report.Distance, 1e-9)
	assert.Equal(t, "1001", report.ID)
	assert.Equal(t, 4*time.Millisecond, report.Latency)
	assert.Equal(t, 6*time.Millisecond, report.MaxLatency)
	assert.Equal(t, 3*time.Millisecond, report.MinLatency)
	assert.Equal(t, time.Millisecond, report.Jitter)
	assert.InDelta(t, 12_500_000, float64(report.DLSpeed), 1e-9)
	assert.InDelta(t, 2_500_000, float64(report.ULSpeed), 1e-9)
	assert.Equal(t, PacketLossStats{Sent: 300, Dup: 1, Max: 301}, report.PacketLoss)
	assert.Equal(
		t,
		TestDuration{
			Ping:     duration(2 * time.Second),
			Download: duration(10 * time.Second),
			Upload:   duration(10 * time.Second),
			Total:    duration(22 * time.Second),
		},
		report.TestDuration,
	)

	empty := Result{Server: server, Ping: nil, Download: nil, Upload: nil, PacketLoss: nil}.Report()

	assert.Equal(
		t,
		TestDuration{Ping: nil, Download: nil, Upload: nil, Total: nil},
		empty.TestDuration,
	)
	assert.Zero(t, empty.DLSpeed)
	assert.Zero(t, empty.Latency)

	data, err := json.Marshal(full)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"distance":6.95,"latency":4000000`)
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// recorder is a [Reporter] that keeps every event, for checking the sequence a run reports.
type recorder struct {
	// events are the events so far.
	events []Event
	// mu guards events, which the race detector checks even though a run serializes its reports.
	mu sync.Mutex
}

// Report keeps the event.
func (r *recorder) Report(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)
}

// names returns the type names of the events, without the live progress events, whose number varies.
func (r *recorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, 0, len(r.events))

	for _, event := range r.events {
		switch event.(type) {
		case LatencyMeasured, TransferProgress:
			continue
		default:
			names = append(names, typeName(event))
		}
	}

	return names
}

// typeName returns an event's type name.
func typeName(event Event) string {
	switch event.(type) {
	case LookupStarted:
		return "LookupStarted"
	case UserFound:
		return "UserFound"
	case ServersFound:
		return "ServersFound"
	case ServerStarted:
		return "ServerStarted"
	case PingStarted:
		return "PingStarted"
	case PingFinished:
		return "PingFinished"
	case PacketLossStarted:
		return "PacketLossStarted"
	case PacketLossFinished:
		return "PacketLossFinished"
	case TransferStarted:
		return "TransferStarted"
	case TransferFinished:
		return "TransferFinished"
	case ServerFinished:
		return "ServerFinished"
	case RunFinished:
		return "RunFinished"
	default:
		return "unknown"
	}
}

// newClient returns a client for the fake API with short transfer tests.
func newClient(t *testing.T, api *testserver.API) *speedtest.Client {
	t.Helper()

	client, err := speedtest.New(
		speedtest.WithBaseURL(api.URL()),
		speedtest.WithTestDuration(200*time.Millisecond),
	)
	require.NoError(t, err)

	return client
}

// serverIDs returns the IDs of servers, in order.
func serverIDs(servers speedtest.Servers) []string {
	ids := make([]string, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, server.ID)
	}

	return ids
}

// TestRunner_Run checks a full run against the fake API: the events arrive in order, every test of the server is
// measured, and packet loss, which the fake API's host cannot measure, is left out of the result.
func TestRunner_Run(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	events := &recorder{events: nil, mu: sync.Mutex{}}

	user, results, err := New(newClient(t, api), events, Settings{}).Run(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []string{
		"LookupStarted", "UserFound", "ServersFound",
		"ServerStarted", "PingStarted", "PingFinished", "PacketLossStarted",
		"TransferStarted", "TransferFinished", "TransferStarted", "TransferFinished",
		"PacketLossFinished", "ServerFinished", "RunFinished",
	}, events.names())

	require.NotNil(t, user)
	assert.Equal(t, testserver.DefaultUser().IP, user.IP)

	require.Len(t, results, 1)
	assert.NotNil(t, results[0].Ping)
	assert.NotNil(t, results[0].Download)
	assert.NotNil(t, results[0].Upload)
	assert.Nil(t, results[0].PacketLoss, "a server that answered no sample measured no packet loss")
}

// TestRunner_Run_NoTransfers checks that skipping both transfers skips their events, and that the packet loss
// measurement ends as soon as it fails instead of waiting out its time.
func TestRunner_Run_NoTransfers(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	events := &recorder{events: nil, mu: sync.Mutex{}}

	start := time.Now()

	_, results, err := New(
		newClient(t, api),
		events,
		Settings{NoDownload: true, NoUpload: true},
	).Run(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []string{
		"LookupStarted", "UserFound", "ServersFound",
		"ServerStarted", "PingStarted", "PingFinished", "PacketLossStarted",
		"PacketLossFinished", "ServerFinished", "RunFinished",
	}, events.names())

	require.Len(t, results, 1)
	assert.Nil(t, results[0].Download)
	assert.Nil(t, results[0].Upload)
	assert.Less(t, time.Since(start), 10*time.Second)
}

// TestRunner_Run_Interrupted checks that a run whose context has ended returns its error and still reports
// RunFinished last.
func TestRunner_Run_Interrupted(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	events := &recorder{events: nil, mu: sync.Mutex{}}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, results, err := New(newClient(t, api), events, Settings{}).Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, results)

	names := events.names()
	require.NotEmpty(t, names)
	assert.Equal(t, "RunFinished", names[len(names)-1])

	finished, ok := events.events[len(events.events)-1].(RunFinished)
	require.True(t, ok)
	require.ErrorIs(t, finished.Err, context.Canceled)
}

// TestRunner_Run_Multi checks that a multi-server run tests the lowest-latency server across the whole pool.
func TestRunner_Run_Multi(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	_, results, err := New(
		newClient(t, api),
		&recorder{events: nil, mu: sync.Mutex{}},
		Settings{Multi: true, NoUpload: true},
	).Run(t.Context())
	require.NoError(t, err)

	require.Len(t, results, 1)
	require.NotNil(t, results[0].Download)

	servers := make([]string, 0, len(results[0].Download.Connections))
	for _, connection := range results[0].Download.Connections {
		servers = append(servers, connection.ServerID)
	}

	assert.Contains(t, servers, results[0].Server.ID, "the main server must take part")
}

// TestLatencyStats checks the summary of loaded round trips, including none.
func TestLatencyStats(t *testing.T) {
	t.Parallel()

	stats := latencyStats(
		[]time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
	)

	assert.Equal(t, 20*time.Millisecond, stats.Latency)
	assert.Equal(t, 10*time.Millisecond, stats.Min)
	assert.Equal(t, 30*time.Millisecond, stats.Max)
	assert.InDelta(
		t,
		float64(8164965*time.Nanosecond),
		float64(stats.Jitter),
		float64(time.Microsecond),
	)
	assert.Len(t, stats.Latencies, 3)

	assert.Equal(
		t,
		speedtest.PingStats{Latencies: []time.Duration{}},
		latencyStats([]time.Duration{}),
	)
}

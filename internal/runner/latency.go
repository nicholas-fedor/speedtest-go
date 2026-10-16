// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// loadedLatency pings a server at a fixed interval while a transfer test loads the connection.
type loadedLatency struct {
	// done is closed when the pings have stopped.
	done chan struct{}
	// latencies are the round trips so far.
	latencies []time.Duration
	// mu guards latencies.
	mu sync.Mutex
}

// loadedLatencyInterval is how often the latency is measured during a transfer test.
const loadedLatencyInterval = 500 * time.Millisecond

// startLoadedLatency starts pinging a server, once per interval, until ctx ends. Each ping uses the tester's ping
// mode, and a ping that fails is left out.
//
// Parameters:
//   - ctx: ends the pings.
//   - tester: sends the pings.
//   - server: the server.
//   - interval: the time between pings.
//
// Returns:
//   - *loadedLatency: the running measurement.
func startLoadedLatency(
	ctx context.Context,
	tester Tester,
	server *speedtest.Server,
	interval time.Duration,
) *loadedLatency {
	loaded := &loadedLatency{done: make(chan struct{}), latencies: nil, mu: sync.Mutex{}}

	go func() {
		defer close(loaded.done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := tester.Ping(
					ctx,
					server,
					speedtest.WithPingCount(1),
					speedtest.WithPingInterval(0),
				)
				if err == nil {
					loaded.mu.Lock()
					loaded.latencies = append(loaded.latencies, stats.Latency)
					loaded.mu.Unlock()
				}
			}
		}
	}()

	return loaded
}

// current returns the latest round trip.
//
// Returns:
//   - time.Duration: the latest round trip, or 0 before the first.
func (l *loadedLatency) current() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.latencies) == 0 {
		return 0
	}

	return l.latencies[len(l.latencies)-1]
}

// wait waits for the pings to stop, after their context ended.
//
// Returns:
//   - []time.Duration: the round trips, in the order they were measured.
func (l *loadedLatency) wait() []time.Duration {
	<-l.done

	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.latencies)
}

// Package echo provides functionality for running periodic ping tests during speed tests.
package echo

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// AccompanyEcho runs periodic ping tests during download/upload to measure latency.
type AccompanyEcho struct {
	// stopEcho stops the ping loop.
	stopEcho chan bool
	// client sends the pings.
	client *speedtest.Client
	// server is the server pinged.
	server *speedtest.Server
	// latencies are the measured latencies in nanoseconds. mu guards them.
	latencies []int64
	// currentLatency is the latest latency in nanoseconds.
	currentLatency atomic.Int64
	// interval is the time between pings.
	interval time.Duration
	// mu guards latencies.
	mu sync.Mutex
}

// New creates a new AccompanyEcho instance, which pings the server with the client's ping mode.
//
// Parameters:
//   - client: the client that sends the pings.
//   - server: the server.
//   - interval: the time between pings.
//
// Returns:
//   - *AccompanyEcho: the echo, which starts pinging on Run.
func New(
	client *speedtest.Client,
	server *speedtest.Server,
	interval time.Duration,
) *AccompanyEcho {
	return &AccompanyEcho{
		client:   client,
		server:   server,
		interval: interval,
		stopEcho: make(chan bool, 1),
	}
}

// Run starts the periodic ping test in a goroutine.
func (ae *AccompanyEcho) Run() {
	ae.latencies = make([]int64, 0)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		ticker := time.NewTicker(ae.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ae.stopEcho:
				return
			case <-ticker.C:
				stats, err := ae.client.Ping(
					ctx,
					ae.server,
					speedtest.WithPingCount(1),
					speedtest.WithPingInterval(0),
				)
				if err == nil {
					latency := int64(stats.Latency)
					ae.currentLatency.Store(latency)
					ae.mu.Lock()
					ae.latencies = append(ae.latencies, latency)
					ae.mu.Unlock()
				}
			}
		}
	}()
}

// Stop stops the periodic ping test.
func (ae *AccompanyEcho) Stop() {
	select {
	case ae.stopEcho <- true:
	default:
	}
}

// CurrentLatency returns the most recent latency measurement.
//
// Returns:
//   - int64: the latency in nanoseconds, or 0 before the first one.
func (ae *AccompanyEcho) CurrentLatency() int64 {
	return ae.currentLatency.Load()
}

// Latencies returns all collected latency measurements.
//
// Returns:
//   - []int64: a copy of the latencies in nanoseconds, or nil before the first one.
func (ae *AccompanyEcho) Latencies() []int64 {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if len(ae.latencies) == 0 {
		return nil
	}

	result := make([]int64, len(ae.latencies))
	copy(result, ae.latencies)

	return result
}

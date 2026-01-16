// Package echo provides functionality for running periodic ping tests during speed tests.
package echo

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// AccompanyEcho runs periodic ping tests during download/upload to measure latency.
type AccompanyEcho struct {
	stopEcho       chan bool
	server         *speedtest.Server
	currentLatency int64
	interval       time.Duration
	latencies      []int64
}

// New creates a new AccompanyEcho instance.
func New(server *speedtest.Server, interval time.Duration) *AccompanyEcho {
	return &AccompanyEcho{
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
				latency, _ := ae.server.HTTPPing(ctx, 1, ae.interval, nil)
				if len(latency) > 0 {
					atomic.StoreInt64(&ae.currentLatency, latency[0])
					ae.latencies = append(ae.latencies, latency[0])
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
func (ae *AccompanyEcho) CurrentLatency() int64 {
	return atomic.LoadInt64(&ae.currentLatency)
}

// Latencies returns all collected latency measurements.
func (ae *AccompanyEcho) Latencies() []int64 {
	return ae.latencies
}

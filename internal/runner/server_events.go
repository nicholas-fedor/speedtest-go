// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// ServerStarted reports that the tests of a server started.
type ServerStarted struct {
	// Server is the server.
	Server *speedtest.Server
}

// PingStarted reports that the latency measurement started.
type PingStarted struct{}

// LatencyMeasured reports one round trip of the latency measurement.
type LatencyMeasured struct {
	// Latency is the round trip.
	Latency time.Duration
}

// PingFinished reports the latency of the server.
type PingFinished struct {
	// Stats are the measured round trips.
	Stats speedtest.PingStats
}

// ServerFinished reports that the tests of a server finished.
type ServerFinished struct {
	// Result is the server's measurements.
	Result speedtest.Result
}

// event marks [ServerStarted] as an event.
func (ServerStarted) event() {}

// event marks [PingStarted] as an event.
func (PingStarted) event() {}

// event marks [LatencyMeasured] as an event.
func (LatencyMeasured) event() {}

// event marks [PingFinished] as an event.
func (PingFinished) event() {}

// event marks [ServerFinished] as an event.
func (ServerFinished) event() {}

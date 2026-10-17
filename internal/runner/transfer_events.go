// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Direction is whether a transfer test downloads or uploads.
type Direction int

// Directions of a transfer test.
const (
	// Download measures the download speed.
	Download Direction = iota
	// Upload measures the upload speed.
	Upload
)

// TransferStarted reports that a transfer test started.
type TransferStarted struct {
	// Direction is whether the test downloads or uploads.
	Direction Direction
}

// TransferProgress reports the live state of a transfer test.
type TransferProgress struct {
	// Direction is whether the test downloads or uploads.
	Direction Direction
	// Rate is the live rate estimate.
	Rate speedtest.ByteRate
	// Latency is the latest round trip measured under the load, or 0 before the first.
	Latency time.Duration
	// Elapsed is the time since the test started measuring.
	Elapsed time.Duration
	// Remaining is the estimated time left, which assumes a settling rate keeps settling.
	Remaining time.Duration
}

// TransferFinished reports the result of a transfer test.
type TransferFinished struct {
	// Latency is the latency measured under the load of the test.
	Latency speedtest.PingStats
	// Transfer is the measurement.
	Transfer speedtest.Transfer
	// Direction is whether the test downloaded or uploaded.
	Direction Direction
}

// String names the direction as the output shows it.
//
// Returns:
//   - string: "Download" or "Upload".
func (d Direction) String() string {
	switch d {
	case Download:
		return "Download"
	case Upload:
		return "Upload"
	default:
		return "Transfer"
	}
}

// event marks [TransferStarted] as an event.
func (TransferStarted) event() {}

// event marks [TransferProgress] as an event.
func (TransferProgress) event() {}

// event marks [TransferFinished] as an event.
func (TransferFinished) event() {}

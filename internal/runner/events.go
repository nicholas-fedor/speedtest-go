// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Event is something a run reports. The events of this package are the only ones.
type Event interface {
	// event marks the types that are events.
	event()
}

// RunFinished is the last event of every run.
type RunFinished struct {
	// Err is the error that ended the run, or nil when every test finished.
	Err error
}

// PacketLossStarted reports that the packet loss measurement started in the background.
type PacketLossStarted struct{}

// PacketLossFinished reports the packet loss measured while the server was tested.
type PacketLossFinished struct {
	// Stats are the counts, with Sent 0 when the server answered none of the samples.
	Stats speedtest.PacketLossStats
}

// event marks [RunFinished] as an event.
func (RunFinished) event() {}

// event marks [PacketLossStarted] as an event.
func (PacketLossStarted) event() {}

// event marks [PacketLossFinished] as an event.
func (PacketLossFinished) event() {}

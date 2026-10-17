// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"bytes"
	"cmp"
	"runtime"
	"sync"
	"time"
)

// Config configures a [Meter].
type Config struct {
	// Logger receives debug messages, or is nil to discard them.
	Logger Logger
	// OnProgress receives the live state of a test at each rate capture, or is nil for none.
	OnProgress func(progress Progress)
	// Connections is the number of workers a download uses and the most an upload uses, or 0 for one per CPU for
	// downloads and up to 8 for uploads.
	Connections int
	// CaptureTime is how long each test runs, or 0 for 15 seconds.
	CaptureTime time.Duration
	// RateCaptureFrequency is how often the rate is sampled, or 0 for every 50 milliseconds.
	RateCaptureFrequency time.Duration
}

// Meter runs one download or upload test and measures it. Each test gets its own meter, so no counts carry over
// between tests.
type Meter struct {
	// logger receives debug messages.
	logger Logger
	// onProgress receives the live state of a test, or is nil.
	onProgress func(progress Progress)
	// download is the download direction.
	download *Direction
	// upload is the upload direction.
	upload *Direction
	// pattern is the data that upload bodies repeat.
	pattern []byte
	// captureTime is how long each test runs.
	captureTime time.Duration
	// rateCaptureFrequency is how often the rate is sampled.
	rateCaptureFrequency time.Duration
	// connections is the number of download workers and the cap on handlers per direction.
	connections int
	// uploadMaxWorkers is the most upload workers that run at once.
	uploadMaxWorkers int
	// cpus bounds the upload workers active when an upload starts.
	cpus int
	// mu guards download and upload.
	mu sync.Mutex
}

// Progress is the live state of a running test.
type Progress struct {
	// Rate is the live rate estimate in bytes per second.
	Rate float64
	// Elapsed is the time since the test started measuring.
	Elapsed time.Duration
	// Remaining is the estimated time left: until the capture time ends, or sooner when the rate is settling and the
	// stability check is expected to stop the test first. It assumes the rate keeps settling, so a test whose rate
	// swings again runs longer than estimated.
	Remaining time.Duration
}

// Meter defaults.
const (
	// defaultCaptureTime is how long a test runs when the configuration sets no capture time.
	defaultCaptureTime = 15 * time.Second
	// defaultRateCaptureFrequency is how often the rate is sampled when the configuration sets no frequency.
	defaultRateCaptureFrequency = 50 * time.Millisecond
	// uploadPatternByte fills upload bodies with alternating bits.
	uploadPatternByte = 0xAA
)

// New returns a meter with the configured worker counts, timing, and rate callback.
//
// Parameters:
//   - cfg: the configuration. Zero fields take their defaults.
//
// Returns:
//   - *Meter: the meter.
func New(cfg Config) *Meter {
	logger := cfg.Logger
	if logger == nil {
		logger = noopLogger{}
	}

	meter := &Meter{
		logger:               logger,
		onProgress:           cfg.OnProgress,
		pattern:              bytes.Repeat([]byte{uploadPatternByte}, readChunkSize),
		captureTime:          cmp.Or(cfg.CaptureTime, defaultCaptureTime),
		rateCaptureFrequency: cmp.Or(cfg.RateCaptureFrequency, defaultRateCaptureFrequency),
		cpus:                 runtime.NumCPU(),
	}

	meter.connections, meter.uploadMaxWorkers = workerLimits(cfg.Connections)
	meter.download = newDirection(meter, kindDownload)
	meter.upload = newDirection(meter, kindUpload)

	return meter
}

// RegisterDownload adds a request handler to the download direction, unless it already has one per connection.
// When the last download test has finished, a new download direction replaces it, so each test starts with its own
// handlers and counts.
//
// Parameters:
//   - handler: sends one download request, counting its data through the given connection.
//
// Returns:
//   - *Direction: the download direction, which runs the handlers.
func (m *Meter) RegisterDownload(handler Handler) *Direction {
	return m.register(&m.download, kindDownload, handler)
}

// RegisterUpload adds a request handler to the upload direction, unless it already has one per connection. When the
// last upload test has finished, a new upload direction replaces it, so each test starts with its own handlers and
// counts.
//
// Parameters:
//   - handler: sends one upload request, counting its data through the given connection.
//
// Returns:
//   - *Direction: the upload direction, which runs the handlers.
func (m *Meter) RegisterUpload(handler Handler) *Direction {
	return m.register(&m.upload, kindUpload, handler)
}

// HandlerLimit returns how many request handlers each direction takes. Handlers registered beyond it are dropped.
//
// Returns:
//   - int: the limit, the number of connections.
func (m *Meter) HandlerLimit() int {
	return m.connections
}

// DownloadConnection returns a connection on the download direction that is not part of a test run, for sending a
// single request outside [Direction.Run].
//
// Returns:
//   - *Connection: the connection.
func (m *Meter) DownloadConnection() *Connection {
	return newConnection(m.current(&m.download), noHandler)
}

// UploadConnection returns a connection on the upload direction that is not part of a test run, for sending a single
// request outside [Direction.Run].
//
// Returns:
//   - *Connection: the connection.
func (m *Meter) UploadConnection() *Connection {
	return newConnection(m.current(&m.upload), noHandler)
}

// register adds a handler to the direction in a slot, unless it already has one per connection. A direction whose
// test has finished is first replaced with a new one.
//
// Parameters:
//   - slot: the meter's download or upload direction, which mu guards.
//   - testKind: the slot's direction of test.
//   - handler: sends one request.
//
// Returns:
//   - *Direction: the direction that holds the handler.
func (m *Meter) register(slot **Direction, testKind kind, handler Handler) *Direction {
	m.mu.Lock()

	if (*slot).stopped.Load() {
		*slot = newDirection(m, testKind)
	}

	direction := *slot
	m.mu.Unlock()

	direction.addHandler(handler, m.connections)

	return direction
}

// current returns the direction in a slot.
//
// Parameters:
//   - slot: the meter's download or upload direction, which mu guards.
//
// Returns:
//   - *Direction: the direction.
func (m *Meter) current(slot **Direction) *Direction {
	m.mu.Lock()
	defer m.mu.Unlock()

	return *slot
}

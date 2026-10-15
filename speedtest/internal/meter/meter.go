// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"bytes"
	"cmp"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Config configures a [Meter].
type Config struct {
	// Logger receives debug messages, or is nil to discard them.
	Logger Logger
	// Connections is the number of workers a download uses and the most an upload uses, or 0 for one per CPU for
	// downloads and up to 8 for uploads.
	Connections int
	// CaptureTime is how long each test runs, or 0 for 15 seconds.
	CaptureTime time.Duration
	// RateCaptureFrequency is how often the rate is sampled, or 0 for every 50 milliseconds.
	RateCaptureFrequency time.Duration
}

// Meter runs the download and upload tests of one client and keeps their totals until [Meter.Reset].
type Meter struct {
	// logger receives debug messages.
	logger Logger
	// download is the current download direction, replaced by Reset.
	download *Direction
	// upload is the current upload direction, replaced by Reset.
	upload *Direction
	// downloadCallback receives the live download rate, or holds nil.
	downloadCallback atomic.Pointer[func(rate float64)]
	// uploadCallback receives the live upload rate, or holds nil.
	uploadCallback atomic.Pointer[func(rate float64)]
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

// Meter defaults.
const (
	// defaultCaptureTime is how long a test runs when the configuration sets no capture time.
	defaultCaptureTime = 15 * time.Second
	// defaultRateCaptureFrequency is how often the rate is sampled when the configuration sets no frequency.
	defaultRateCaptureFrequency = 50 * time.Millisecond
	// uploadPatternByte fills upload bodies with alternating bits.
	uploadPatternByte = 0xAA
)

// New returns a meter with the configured worker counts and timing.
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

// SetCallbackDownload sets the function that receives the live download rate during each download test. It stays
// set across [Meter.Reset].
//
// Parameters:
//   - callback: receives the rate in bytes per second, or nil for none.
func (m *Meter) SetCallbackDownload(callback func(rate float64)) {
	m.downloadCallback.Store(callbackPointer(callback))
}

// SetCallbackUpload sets the function that receives the live upload rate during each upload test. It stays set
// across [Meter.Reset].
//
// Parameters:
//   - callback: receives the rate in bytes per second, or nil for none.
func (m *Meter) SetCallbackUpload(callback func(rate float64)) {
	m.uploadCallback.Store(callbackPointer(callback))
}

// RegisterDownload adds a request handler to the download direction, unless it already has one per connection.
// When the last download test has finished, a new download direction replaces it, so each test starts with its own
// handlers and totals.
//
// Parameters:
//   - handler: sends one download request.
//
// Returns:
//   - *Direction: the download direction, which runs the handlers.
func (m *Meter) RegisterDownload(handler func()) *Direction {
	return m.register(&m.download, kindDownload, handler)
}

// RegisterUpload adds a request handler to the upload direction, unless it already has one per connection. When the
// last upload test has finished, a new upload direction replaces it, so each test starts with its own handlers and
// totals.
//
// Parameters:
//   - handler: sends one upload request.
//
// Returns:
//   - *Direction: the upload direction, which runs the handlers.
func (m *Meter) RegisterUpload(handler func()) *Direction {
	return m.register(&m.upload, kindUpload, handler)
}

// NewChunk returns a chunk that counts one request's data toward this meter.
//
// Returns:
//   - *Chunk: the chunk, ready for [Chunk.DownloadHandler] or [Chunk.UploadHandler].
func (m *Meter) NewChunk() *Chunk {
	return &Chunk{meter: m}
}

// AddTotalUpload counts upload bytes that the server confirmed.
//
// Parameters:
//   - value: the confirmed bytes.
func (m *Meter) AddTotalUpload(value int64) {
	m.currentUpload().confirmed.Add(value)
}

// TotalDownload returns the bytes downloaded since the last reset.
//
// Returns:
//   - int64: the downloaded bytes.
func (m *Meter) TotalDownload() int64 {
	return m.currentDownload().confirmed.Load()
}

// TotalUpload returns the upload bytes the server confirmed since the last reset.
//
// Returns:
//   - int64: the confirmed upload bytes.
func (m *Meter) TotalUpload() int64 {
	return m.currentUpload().confirmed.Load()
}

// UploadBacklog returns the upload bytes read into the transport but not yet confirmed by a successful response.
//
// Returns:
//   - int64: the unconfirmed bytes, never negative.
func (m *Meter) UploadBacklog() int64 {
	upload := m.currentUpload()

	return max(upload.read.Load()-upload.confirmed.Load(), 0)
}

// UploadConfirmationRatio returns the share of upload bytes read into the transport that the server confirmed. It
// counts application-level confirmations, not TCP acknowledgments.
//
// Returns:
//   - float64: the share from 0 to 1, or 0 before any bytes were read.
func (m *Meter) UploadConfirmationRatio() float64 {
	upload := m.currentUpload()

	read := upload.read.Load()
	if read <= 0 {
		return 0
	}

	return min(max(float64(upload.confirmed.Load())/float64(read), 0), 1)
}

// Reset starts new download and upload directions, so the totals start again from zero.
func (m *Meter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.download = newDirection(m, kindDownload)
	m.upload = newDirection(m, kindUpload)
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
func (m *Meter) register(slot **Direction, testKind kind, handler func()) *Direction {
	m.mu.Lock()

	if (*slot).stopped.Load() {
		*slot = newDirection(m, testKind)
	}

	direction := *slot
	m.mu.Unlock()

	direction.addHandler(handler, m.connections)

	return direction
}

// currentDownload returns the current download direction.
//
// Returns:
//   - *Direction: the direction.
func (m *Meter) currentDownload() *Direction {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.download
}

// currentUpload returns the current upload direction.
//
// Returns:
//   - *Direction: the direction.
func (m *Meter) currentUpload() *Direction {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.upload
}

// callbackPointer returns a pointer to a callback for atomic storage.
//
// Parameters:
//   - callback: the callback, or nil.
//
// Returns:
//   - *func(rate float64): a pointer to callback, or nil for a nil callback.
func callbackPointer(callback func(rate float64)) *func(rate float64) {
	if callback == nil {
		return nil
	}

	return &callback
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// recordingLogger records the messages and arguments of the meter's debug messages.
type recordingLogger struct {
	// messages maps each message to the arguments it was last logged with. mu guards it.
	messages map[string][]any
	// mu guards messages.
	mu sync.Mutex
}

// zeroReader reads endless zero bytes.
type zeroReader struct{}

// newTestMeter returns a meter with a connection count and a capture time, and the other defaults.
//
// Parameters:
//   - connections: the connection count.
//   - captureTime: how long each test runs.
//
// Returns:
//   - *Meter: the meter.
func newTestMeter(connections int, captureTime time.Duration) *Meter {
	return New(Config{
		Logger:               nil,
		OnRate:               nil,
		Connections:          connections,
		CaptureTime:          captureTime,
		RateCaptureFrequency: 0,
	})
}

// idle is a handler that sends nothing and succeeds.
//
// Returns:
//   - error: always nil.
func idle(_ *Connection) error {
	return nil
}

// Debug records the message and its arguments.
//
// Parameters:
//   - msg: the message.
//   - args: the message's keys and values.
func (l *recordingLogger) Debug(_ context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.messages[msg] = args
}

// Read fills buffer with zero bytes.
//
// Parameters:
//   - buffer: the destination.
//
// Returns:
//   - int: the length of buffer.
//   - error: always nil.
func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)

	return len(buffer), nil
}

// TestNew checks the defaults and that the configuration reaches the meter.
func TestNew(t *testing.T) {
	t.Parallel()

	defaults := New(Config{})

	assert.Equal(t, defaultCaptureTime, defaults.captureTime)
	assert.Equal(t, defaultRateCaptureFrequency, defaults.rateCaptureFrequency)
	assert.Equal(t, runtime.NumCPU(), defaults.connections)
	assert.Equal(t, defaultUploadMaxWorkers, defaults.uploadMaxWorkers)
	assert.IsType(t, noopLogger{}, defaults.logger)
	assert.Nil(t, defaults.onRate)

	logger := &recordingLogger{messages: map[string][]any{}}
	configured := New(Config{
		Logger:               logger,
		OnRate:               func(float64) {},
		Connections:          3,
		CaptureTime:          time.Second,
		RateCaptureFrequency: 10 * time.Millisecond,
	})

	assert.Same(t, logger, configured.logger)
	assert.NotNil(t, configured.onRate)
	assert.Equal(t, 3, configured.connections)
	assert.Equal(t, 3, configured.uploadMaxWorkers)
	assert.Equal(t, time.Second, configured.captureTime)
	assert.Equal(t, 10*time.Millisecond, configured.rateCaptureFrequency)
}

// TestMeter_Register checks that each direction takes handlers up to one per connection, also when they are
// registered concurrently.
func TestMeter_Register(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(4, 0)

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			meter.RegisterDownload(idle)
			meter.RegisterUpload(idle)
		})
	}

	wg.Wait()

	assert.Same(t, meter.download, meter.RegisterDownload(idle))
	assert.Same(t, meter.upload, meter.RegisterUpload(idle))
	assert.Len(t, meter.download.snapshotHandlers(), 4)
	assert.Len(t, meter.upload.snapshotHandlers(), 4)
}

// TestMeter_Connections checks that a connection outside a test run counts toward its own direction, and that it
// runs no registered handler.
func TestMeter_Connections(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(1, 0)

	download := meter.DownloadConnection()
	upload := meter.UploadConnection()

	assert.Same(t, meter.download, download.direction)
	assert.Same(t, meter.upload, upload.direction)
	assert.Equal(t, noHandler, download.handler)
	assert.Equal(t, noHandler, upload.handler)
}

// TestMeter_HandlerLimit checks that the handler limit is the connection count.
func TestMeter_HandlerLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, newTestMeter(3, 0).HandlerLimit())
	assert.Equal(t, runtime.NumCPU(), New(Config{}).HandlerLimit())
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return New(
		Config{
			Logger:               nil,
			Connections:          connections,
			CaptureTime:          captureTime,
			RateCaptureFrequency: 0,
		},
	)
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

// TestNew checks the defaults and that the configuration reaches the meter.
func TestNew(t *testing.T) {
	t.Parallel()

	defaults := New(Config{})

	assert.Equal(t, defaultCaptureTime, defaults.captureTime)
	assert.Equal(t, defaultRateCaptureFrequency, defaults.rateCaptureFrequency)
	assert.Equal(t, runtime.NumCPU(), defaults.connections)
	assert.Equal(t, defaultUploadMaxWorkers, defaults.uploadMaxWorkers)
	assert.IsType(t, noopLogger{}, defaults.logger)

	logger := &recordingLogger{messages: map[string][]any{}}
	configured := New(Config{
		Logger:               logger,
		Connections:          3,
		CaptureTime:          time.Second,
		RateCaptureFrequency: 10 * time.Millisecond,
	})

	assert.Same(t, logger, configured.logger)
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
			meter.RegisterDownload(func() {})
			meter.RegisterUpload(func() {})
		})
	}

	wg.Wait()

	assert.Same(t, meter.download, meter.RegisterDownload(func() {}))
	assert.Same(t, meter.upload, meter.RegisterUpload(func() {}))
	assert.Len(t, meter.download.snapshotHandlers(), 4)
	assert.Len(t, meter.upload.snapshotHandlers(), 4)
}

// TestMeter_Totals checks that downloads and confirmed uploads add to their totals, and that Reset clears them.
func TestMeter_Totals(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(1, 0)

	require.NoError(t, meter.NewChunk().DownloadHandler(io.LimitReader(zeroReader{}, 5000)))
	meter.AddTotalUpload(300)

	assert.Equal(t, int64(5000), meter.TotalDownload())
	assert.Equal(t, int64(300), meter.TotalUpload())

	meter.Reset()

	assert.Zero(t, meter.TotalDownload())
	assert.Zero(t, meter.TotalUpload())
}

// TestMeter_UploadBacklog checks the backlog and confirmation ratio as chunks are read and confirmed.
func TestMeter_UploadBacklog(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(1, 0)
	assert.Zero(t, meter.UploadConfirmationRatio(), "nothing read yet")

	_, err := io.Copy(io.Discard, meter.NewChunk().UploadHandler(3*readChunkSize))
	require.NoError(t, err)

	assert.Equal(t, int64(3*readChunkSize), meter.UploadBacklog())
	assert.Zero(t, meter.UploadConfirmationRatio())

	meter.AddTotalUpload(readChunkSize)
	assert.Equal(t, int64(2*readChunkSize), meter.UploadBacklog())
	assert.InDelta(t, 1.0/3, meter.UploadConfirmationRatio(), 1e-9)

	meter.AddTotalUpload(5 * readChunkSize)
	assert.Zero(t, meter.UploadBacklog(), "the backlog never goes negative")
	assert.InDelta(t, 1.0, meter.UploadConfirmationRatio(), 1e-9, "the ratio never exceeds 1")
}

// TestMeter_ConcurrentReset checks that resets race safely with reads of the totals.
func TestMeter_ConcurrentReset(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(1, 0)

	var wg sync.WaitGroup

	for index := range 100 {
		wg.Go(meter.Reset)

		if index%10 == 0 {
			wg.Go(func() {
				_ = meter.TotalDownload()
				_ = meter.TotalUpload()
			})
		}
	}

	wg.Wait()
}

// TestMeter_Callbacks checks that a callback set before a reset still receives the rates of later tests, and that a
// nil callback clears it.
func TestMeter_Callbacks(t *testing.T) {
	t.Parallel()

	meter := newTestMeter(1, 0)

	var got []float64

	meter.SetCallbackDownload(func(rate float64) { got = append(got, rate) })
	meter.Reset()

	callback := meter.currentDownload().callback()
	require.NotNil(t, callback)
	callback(42)
	assert.Equal(t, []float64{42}, got)

	assert.Nil(t, meter.currentUpload().callback(), "no upload callback was set")

	meter.SetCallbackDownload(nil)
	assert.Nil(t, meter.currentDownload().callback())
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

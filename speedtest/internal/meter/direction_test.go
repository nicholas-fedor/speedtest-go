// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Errors of the test handlers.
var (
	// errRequest is the error of a request that fails.
	errRequest = errors.New("request failed")
)

// confirming returns a handler that waits and then counts bytes toward its connection.
//
// Parameters:
//   - wait: how long each request takes.
//   - bytes: the bytes each request counts.
//
// Returns:
//   - Handler: the handler.
func confirming(wait time.Duration, bytes int64) Handler {
	return func(connection *Connection) error {
		time.Sleep(wait)
		connection.Confirm(bytes)

		return nil
	}
}

// runTest runs a direction until it stops, with a cancel function that does nothing else.
//
// Parameters:
//   - direction: the direction.
func runTest(direction *Direction) {
	_, cancel := context.WithCancel(context.Background())
	direction.Run(context.Background(), cancel, 0)
}

// TestDirection_RunStopsWithContext checks that a test stops when its context ends, before the capture time, and
// that its workers and chunks then see it as stopped.
func TestDirection_RunStopsWithContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		direction := newTestMeter(2, time.Hour).RegisterDownload(confirming(time.Millisecond, 0))

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()

		direction.Run(ctx, cancel, 0)

		assert.Less(t, time.Since(start), time.Second)
		assert.True(t, direction.stopped.Load())
		assert.False(t, direction.waitForTurn(0))
	})
}

// TestDirection_Result checks a multi-handler test's result: the main handler's workers come first, the others share
// the remaining handlers, every connection's bytes add up to the total, failed requests are counted, and each
// connection's active time stays within the capture window.
func TestDirection_Result(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		meter := newTestMeter(4, 1050*time.Millisecond)
		meter.RegisterDownload(confirming(100*time.Millisecond, 1_000))
		meter.RegisterDownload(func(*Connection) error {
			time.Sleep(100 * time.Millisecond)

			return errRequest
		})
		direction := meter.RegisterDownload(confirming(100*time.Millisecond, 2_000))

		runTest(direction)

		result := direction.Result()

		handlers := make([]int, 0, len(result.Connections))

		var (
			bytes    int64
			requests int
			failures int
		)

		for _, connection := range result.Connections {
			handlers = append(handlers, connection.Handler)
			bytes += connection.Bytes
			requests += connection.Requests
			failures += connection.Failures

			assert.LessOrEqual(t, connection.Active, result.Window)
			assert.Positive(t, connection.Active)
		}

		assert.Equal(t, []int{0, 1, 2, 1}, handlers)
		assert.Equal(t, result.Bytes, bytes, "the connections' bytes add up to the total")
		assert.Equal(t, int64(10*(1_000+2_000)), result.Bytes)
		assert.InDelta(t, float64(result.Bytes)/1.05, result.Rate, 1e-6)
		assert.Equal(t, result.Requests, requests)
		assert.Equal(t, result.Failures, failures)
		assert.Equal(
			t,
			2*11,
			result.Failures,
			"each failing worker sends ten requests and one cut short",
		)
		assert.Zero(t, result.Sent, "a download sends nothing")
	})
}

// TestDirection_ResultSent checks that an upload's result reports the bytes read into the transport by the stop.
func TestDirection_ResultSent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		direction := newTestMeter(
			1,
			250*time.Millisecond,
		).RegisterUpload(func(connection *Connection) error {
			time.Sleep(100 * time.Millisecond)

			_, err := connection.NewChunk().
				UploadHandler(readChunkSize).
				Read(make([]byte, readChunkSize))
			connection.Confirm(readChunkSize / 2)

			return err
		})

		runTest(direction)

		result := direction.Result()
		assert.Equal(t, int64(2*readChunkSize), result.Sent)
		assert.Equal(t, int64(readChunkSize), result.Bytes)
	})
}

// TestMeter_RegisterAfterTest checks that a second test on the same meter, without a reset, runs its own handlers
// and reports its own rate, while its earlier direction keeps the first test's results.
func TestMeter_RegisterAfterTest(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		meter := newTestMeter(1, 1050*time.Millisecond)

		first := meter.RegisterDownload(confirming(100*time.Millisecond, 1_000))
		runTest(first)

		second := meter.RegisterDownload(confirming(100*time.Millisecond, 2_000))
		runTest(second)

		require.NotSame(t, first, second, "a finished direction must not be reused")
		assert.InDelta(t, 10_000/1.05, first.FinalRate(), 1e-6)
		assert.InDelta(t, 20_000/1.05, second.FinalRate(), 1e-6)
		assert.Len(t, second.snapshotHandlers(), 1, "the new direction holds only the new handler")
	})
}

// TestDirection_RunLogsWorkers checks that a test reports how its workers are shared to the meter's logger.
func TestDirection_RunLogsWorkers(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		logger := &recordingLogger{messages: map[string][]any{}}
		meter := New(Config{
			Logger:               logger,
			OnRate:               nil,
			Connections:          2,
			CaptureTime:          100 * time.Millisecond,
			RateCaptureFrequency: 0,
		})

		runTest(meter.RegisterDownload(confirming(10*time.Millisecond, 0)))

		logger.mu.Lock()
		defer logger.mu.Unlock()

		assert.Equal(
			t,
			[]any{logKeyHandlers, 1, logKeyMainWorkers, 2, logKeyOtherWorkers, 0},
			logger.messages["starting transfer workers"],
		)
		assert.Contains(t, logger.messages, "stopped transfer workers")
	})
}

// TestDirection_OnRate checks that the configured callback receives the live rate while the test runs.
func TestDirection_OnRate(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var (
			mu    sync.Mutex
			rates []float64
		)

		meter := New(Config{
			Logger: nil,
			OnRate: func(rate float64) {
				mu.Lock()
				defer mu.Unlock()

				rates = append(rates, rate)
			},
			Connections:          1,
			CaptureTime:          500 * time.Millisecond,
			RateCaptureFrequency: 0,
		})

		runTest(meter.RegisterDownload(confirming(10*time.Millisecond, 1_000)))

		mu.Lock()
		defer mu.Unlock()

		require.NotEmpty(t, rates)
		assert.Positive(t, rates[len(rates)-1])
	})
}

// TestDirection_rateCaptureStops checks that the capture goroutine ends when its stop channel closes. The bubble
// fails the test if it is still running.
func TestDirection_rateCaptureStops(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(_ *testing.T) {
		direction := newDirection(newTestMeter(1, 0), kindDownload)
		stop := make(chan struct{})
		direction.rateCapture(time.Now(), stop)

		time.Sleep(3 * defaultRateCaptureFrequency)
		close(stop)
	})
}

// TestRateSample checks that downloads sample the period's own volume and uploads sample the cumulative average
// scaled to one period, also in periods without new bytes.
func TestRateSample(t *testing.T) {
	t.Parallel()

	const frequency = 50 * time.Millisecond

	tests := []struct {
		name      string
		testKind  kind
		globalAvg float64
		delta     int64
		want      float64
	}{
		{
			name:      "download uses the delta",
			testKind:  kindDownload,
			globalAvg: 2000,
			delta:     300,
			want:      300,
		},
		{name: "download idle period", testKind: kindDownload, globalAvg: 2000, delta: 0, want: 0},
		{
			name:      "upload uses the average",
			testKind:  kindUpload,
			globalAvg: 2000,
			delta:     300,
			want:      100,
		},
		{name: "upload idle period", testKind: kindUpload, globalAvg: 2000, delta: 0, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.InDelta(
				t,
				tt.want,
				rateSample(tt.testKind, tt.globalAvg, tt.delta, frequency),
				1e-9,
			)
		})
	}
}

// TestDirection_rateCaptureBursts feeds a whole chunk every other capture period, the pattern confirmed uploads
// produce, and checks the reported rate once it has settled over the first half of the periods. Uploads then rise
// steadily instead of dipping on the periods without new bytes. Downloads, which sample each period's own volume,
// still dip on every such period, which shows the pattern tells the two apart.
func TestDirection_rateCaptureBursts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testKind kind
		wantDips int
	}{
		{name: "upload", testKind: kindUpload, wantDips: 0},
		{name: "download", testKind: kindDownload, wantDips: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				var (
					mu    sync.Mutex
					rates []float64
				)

				meter := New(Config{
					Logger: nil,
					OnRate: func(rate float64) {
						mu.Lock()
						defer mu.Unlock()

						rates = append(rates, rate)
					},
					Connections:          1,
					CaptureTime:          0,
					RateCaptureFrequency: 0,
				})
				direction := newDirection(meter, tt.testKind)
				connection := newConnection(direction, 0)
				direction.connections = []*Connection{connection}

				stop := make(chan struct{})
				direction.rateCapture(time.Now(), stop)

				// Bursts land halfway between ticks, so no burst races a tick.
				time.Sleep(defaultRateCaptureFrequency / 2)

				for period := range 20 {
					if period%2 == 0 {
						connection.Confirm(64 * 1024)
					}

					time.Sleep(defaultRateCaptureFrequency)
					synctest.Wait()
				}

				close(stop)

				mu.Lock()
				defer mu.Unlock()

				require.Len(t, rates, 20)

				dips := 0

				for index := len(rates) / 2; index < len(rates); index++ {
					if rates[index] < rates[index-1] {
						dips++
					}
				}

				assert.Equal(t, tt.wantDips, dips, "settled rates: %v", rates[len(rates)/2:])
			})
		})
	}
}

// TestMeasuredRate checks the rate arithmetic, including a window with no elapsed time.
func TestMeasuredRate(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1_500_000.0, measuredRate(3_000_000, 2*time.Second), 1e-9)
	assert.Zero(t, measuredRate(3_000_000, 0))
	assert.Zero(t, measuredRate(3_000_000, -time.Second))
}

// TestDirection_FinalRate checks that the final rate is the measured volume over the capture window. One megabyte
// arrives every 100 ms, ten times in a 1.05 s window, for both directions. The live estimate starts at zero and would
// still read well below this when the window ends.
func TestDirection_FinalRate(t *testing.T) {
	t.Parallel()

	for _, testKind := range []kind{kindDownload, kindUpload} {
		synctest.Test(t, func(t *testing.T) {
			direction := newDirection(newTestMeter(1, 1050*time.Millisecond), testKind)
			direction.addHandler(confirming(100*time.Millisecond, 1_000_000), 1)

			runTest(direction)

			assert.InDelta(t, 10_000_000/1.05, direction.FinalRate(), 1e-6)

			// The capture can still be finishing a tick after the stop, so read the live estimate under its lock.
			direction.welfordMu.Lock()
			ewma := direction.welford.EWMA()
			direction.welfordMu.Unlock()

			assert.Less(t, ewma, 0.9*direction.FinalRate(), "the live estimate is still converging")
		})
	}
}

// TestDirection_FinalRateExcludesCancellation checks that bytes arriving while requests are cancelled, after the
// capture stops, are left out of the result.
func TestDirection_FinalRateExcludesCancellation(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		direction := newDirection(newTestMeter(1, 100*time.Millisecond), kindDownload)

		var once sync.Once

		direction.addHandler(func(connection *Connection) error {
			once.Do(func() { connection.Confirm(1_000) })
			time.Sleep(10 * time.Millisecond)

			return nil
		}, 1)

		direction.Run(
			context.Background(),
			func() { direction.connections[0].Confirm(1_000_000) },
			0,
		)

		assert.Equal(t, int64(1_001_000), direction.volume())
		assert.Equal(t, int64(1_000), direction.Result().Bytes)
		assert.InDelta(
			t,
			10_000.0,
			direction.FinalRate(),
			1e-9,
			"only the 1,000 bytes before the stop count",
		)
	})
}

// TestDirection_FinalRateIgnoresBlockedCallback checks that a rate callback that blocks neither delays the stop nor
// changes the final rate: the test stops on time, and bytes that arrive while the capture is still busy are left
// out, whichever case the capture handles next.
func TestDirection_FinalRateIgnoresBlockedCallback(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		meter := New(Config{
			Logger:               nil,
			OnRate:               func(float64) { <-release },
			Connections:          1,
			CaptureTime:          100 * time.Millisecond,
			RateCaptureFrequency: 0,
		})
		direction := newDirection(meter, kindDownload)

		var once sync.Once

		direction.addHandler(func(connection *Connection) error {
			once.Do(func() { connection.Confirm(1_000) })
			time.Sleep(10 * time.Millisecond)

			return nil
		}, 1)

		start := time.Now()

		runTest(direction)

		// The stop comes at 100 ms. A worker that started another 10 ms request at that same instant finishes it first.
		assert.LessOrEqual(
			t,
			time.Since(start),
			110*time.Millisecond,
			"a blocked callback must not delay the stop",
		)

		direction.connections[0].Confirm(1_000_000)
		close(release)

		assert.InDelta(
			t,
			10_000.0,
			direction.FinalRate(),
			1e-9,
			"only the 1,000 bytes before the stop count",
		)
	})
}

// TestDirection_RunWithoutHandlers checks that running a direction with no handlers is a programming error.
func TestDirection_RunWithoutHandlers(t *testing.T) {
	t.Parallel()

	direction := newDirection(newTestMeter(1, 0), kindDownload)

	assert.Panics(t, func() { direction.Run(context.Background(), func() {}, 0) })
}

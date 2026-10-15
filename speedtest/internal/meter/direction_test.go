// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirection_RunStopsWithContext checks that a test stops when its context ends, before the capture time, and
// that its workers and chunks then see it as stopped.
func TestDirection_RunStopsWithContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		meter := newTestMeter(2, time.Hour)
		direction := meter.RegisterDownload(func() { time.Sleep(time.Millisecond) })

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()

		direction.Run(ctx, cancel, 0)

		assert.Less(t, time.Since(start), time.Second)
		assert.True(t, direction.stopped.Load())
		assert.False(t, direction.waitForTurn(0))
	})
}

// TestMeter_RegisterAfterTest checks that a second test on the same meter, without a reset, runs its own handlers
// and reports its own rate, while its earlier direction keeps the first test's results.
func TestMeter_RegisterAfterTest(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		meter := newTestMeter(1, 1050*time.Millisecond)

		run := func(bytes int64) *Direction {
			var direction *Direction

			direction = meter.RegisterDownload(func() {
				time.Sleep(100 * time.Millisecond)
				direction.confirmed.Add(bytes)
			})

			_, cancel := context.WithCancel(context.Background())
			direction.Run(context.Background(), cancel, 0)

			return direction
		}

		first := run(1_000)
		second := run(2_000)

		require.NotSame(t, first, second, "a finished direction must not be reused")
		assert.InDelta(t, 10_000/1.05, first.FinalRate(), 1e-6)
		assert.InDelta(t, 20_000/1.05, second.FinalRate(), 1e-6)
		assert.Len(t, second.snapshotHandlers(), 1, "the new direction holds only the new handler")
		assert.Equal(
			t,
			second.confirmed.Load(),
			meter.TotalDownload(),
			"the totals are the latest test's",
		)
	})
}

// TestDirection_RunLogsWorkers checks that a test reports how its workers are shared to the meter's logger.
func TestDirection_RunLogsWorkers(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		logger := &recordingLogger{messages: map[string][]any{}}
		meter := New(
			Config{
				Logger:               logger,
				Connections:          2,
				CaptureTime:          100 * time.Millisecond,
				RateCaptureFrequency: 0,
			},
		)
		direction := meter.RegisterDownload(func() { time.Sleep(10 * time.Millisecond) })

		_, cancel := context.WithCancel(context.Background())
		direction.Run(context.Background(), cancel, 0)

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

// TestDirection_CallbackDuringRun checks that the live rate reaches a callback set while the test runs.
func TestDirection_CallbackDuringRun(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		meter := newTestMeter(1, 500*time.Millisecond)
		direction := meter.RegisterDownload(func() {
			time.Sleep(10 * time.Millisecond)
			meter.currentDownload().confirmed.Add(1000)
		})

		var (
			mu    sync.Mutex
			rates int
		)

		go func() {
			time.Sleep(100 * time.Millisecond)
			meter.SetCallbackDownload(func(float64) {
				mu.Lock()
				defer mu.Unlock()

				rates++
			})
		}()

		_, cancel := context.WithCancel(context.Background())
		direction.Run(context.Background(), cancel, 0)

		mu.Lock()
		defer mu.Unlock()

		assert.Positive(t, rates)
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
				meter := newTestMeter(1, 0)
				direction := newDirection(meter, tt.testKind)

				var (
					mu    sync.Mutex
					rates []float64
				)

				record := func(rate float64) {
					mu.Lock()
					defer mu.Unlock()

					rates = append(rates, rate)
				}

				if tt.testKind == kindUpload {
					meter.SetCallbackUpload(record)
				} else {
					meter.SetCallbackDownload(record)
				}

				stop := make(chan struct{})
				direction.rateCapture(time.Now(), stop)

				// Bursts land halfway between ticks, so no burst races a tick.
				time.Sleep(defaultRateCaptureFrequency / 2)

				for period := range 20 {
					if period%2 == 0 {
						direction.confirmed.Add(64 * 1024)
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
			direction.addHandler(func() {
				time.Sleep(100 * time.Millisecond)
				direction.confirmed.Add(1_000_000)
			}, 1)

			_, cancel := context.WithCancel(context.Background())
			direction.Run(context.Background(), cancel, 0)

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
// capture stops, are left out of the final rate.
func TestDirection_FinalRateExcludesCancellation(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		direction := newDirection(newTestMeter(1, 100*time.Millisecond), kindDownload)

		var once sync.Once

		direction.addHandler(func() {
			once.Do(func() { direction.confirmed.Add(1_000) })
			time.Sleep(10 * time.Millisecond)
		}, 1)

		direction.Run(context.Background(), func() { direction.confirmed.Add(1_000_000) }, 0)

		assert.Equal(t, int64(1_001_000), direction.confirmed.Load())
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
		meter := newTestMeter(1, 100*time.Millisecond)
		direction := newDirection(meter, kindDownload)
		release := make(chan struct{})

		meter.SetCallbackDownload(func(float64) { <-release })

		var once sync.Once

		direction.addHandler(func() {
			once.Do(func() { direction.confirmed.Add(1_000) })
			time.Sleep(10 * time.Millisecond)
		}, 1)

		start := time.Now()

		_, cancel := context.WithCancel(context.Background())
		direction.Run(context.Background(), cancel, 0)

		// The stop comes at 100 ms. A worker that started another 10 ms request at that same instant finishes it first.
		assert.LessOrEqual(
			t,
			time.Since(start),
			110*time.Millisecond,
			"a blocked callback must not delay the stop",
		)

		direction.confirmed.Add(1_000_000)
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

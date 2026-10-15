// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mebibyte is one MiB, the unit of the backlog limit.
const mebibyte = 1024 * 1024

// TestNextUploadWorkers checks each decision: growth on a better rate, holding steady, proportional drops for a
// lower rate or a large backlog, and the bounds of one worker and the cap.
func TestNextUploadWorkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		active     int
		rate       float64
		bestRate   float64
		backlog    int64
		wantActive int
		wantBest   float64
	}{
		{
			name:       "first measurement grows",
			active:     2,
			rate:       100,
			bestRate:   0,
			wantActive: 3,
			wantBest:   100,
		},
		{
			name:       "better rate grows",
			active:     3,
			rate:       110,
			bestRate:   100,
			wantActive: 4,
			wantBest:   110,
		},
		{
			name:       "growth stops at the cap",
			active:     8,
			rate:       200,
			bestRate:   100,
			wantActive: 8,
			wantBest:   200,
		},
		{
			name:       "similar rate holds",
			active:     4,
			rate:       102,
			bestRate:   100,
			wantActive: 4,
			wantBest:   100,
		},
		{name: "rate halves", active: 8, rate: 50, bestRate: 100, wantActive: 4, wantBest: 50},
		{
			name:       "slight drop removes one",
			active:     4,
			rate:       84,
			bestRate:   100,
			wantActive: 3,
			wantBest:   84,
		},
		{
			name:       "backlog twice the limit halves",
			active:     4,
			rate:       100,
			bestRate:   100,
			backlog:    16 * mebibyte,
			wantActive: 2,
			wantBest:   100,
		},
		{
			name:       "one worker stays",
			active:     1,
			rate:       10,
			bestRate:   100,
			wantActive: 1,
			wantBest:   100,
		},
		{name: "never below one", active: 2, rate: 1, bestRate: 100, wantActive: 1, wantBest: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			active, best := nextUploadWorkers(tt.active, 8, tt.rate, tt.bestRate, tt.backlog)
			assert.Equal(t, tt.wantActive, active)
			assert.InDelta(t, tt.wantBest, best, 1e-9)
		})
	}
}

// TestAdaptUploadWorkers drives the controller loop with fake confirmed and read volumes, one adjustment per
// interval: growth while the rate improves, holding while it is steady, then drops for a lower rate and a backlog.
func TestAdaptUploadWorkers(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		td := newDirection(New(Config{}), kindUpload)
		td.maxWorkers = 8
		td.activeWorkers = 2

		stop := make(chan struct{})

		var done sync.WaitGroup

		done.Go(func() { td.adaptUploadWorkers(context.Background(), stop) })

		// Volumes change halfway between adjustments, so no change races a tick.
		time.Sleep(uploadAdjustInterval / 2)

		steps := []struct {
			confirmed int64
			backlog   int64
		}{
			{confirmed: 1_000_000},
			{confirmed: 2_000_000},
			{confirmed: 2_000_000},
			{confirmed: 1_000_000},
			{confirmed: 1_000_000, backlog: 20 * mebibyte},
		}

		got := make([]int, 0, len(steps))

		for _, step := range steps {
			td.confirmed.Add(step.confirmed)
			td.read.Add(step.confirmed + step.backlog)

			time.Sleep(uploadAdjustInterval)
			synctest.Wait()

			td.workerMu.Lock()
			got = append(got, td.activeWorkers)
			td.workerMu.Unlock()
		}

		close(stop)
		done.Wait()

		assert.Equal(t, []int{3, 4, 4, 2, 1}, got)
	})
}

// TestAdaptUploadWorkersStalled checks intervals without new confirmations: before the first confirmation they are
// skipped, a quiet interval with a small backlog keeps the workers, and one with a large backlog removes them.
func TestAdaptUploadWorkersStalled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		td := newDirection(New(Config{}), kindUpload)
		td.maxWorkers = 8
		td.activeWorkers = 4

		stop := make(chan struct{})

		var done sync.WaitGroup

		done.Go(func() { td.adaptUploadWorkers(context.Background(), stop) })

		time.Sleep(uploadAdjustInterval / 2)

		steps := []struct {
			confirmed int64
			read      int64
		}{
			{confirmed: 0, read: 4 * mebibyte},
			{confirmed: 1_000_000, read: 1_000_000},
			{confirmed: 0, read: mebibyte},
			{confirmed: 0, read: 30 * mebibyte},
		}

		got := make([]int, 0, len(steps))

		for _, step := range steps {
			td.confirmed.Add(step.confirmed)
			td.read.Add(step.read)

			time.Sleep(uploadAdjustInterval)
			synctest.Wait()

			td.workerMu.Lock()
			got = append(got, td.activeWorkers)
			td.workerMu.Unlock()
		}

		close(stop)
		done.Wait()

		// Skipped before any confirmation, grows on the first, holds through a quiet interval with a 5 MiB backlog
		// under the 10 MiB limit, then drops for a 35 MiB backlog.
		assert.Equal(t, []int{4, 5, 5, 2}, got)
	})
}

// TestWaitForTurn checks that a worker above the active count waits, resumes when the count rises, and gives up
// when the test stops.
func TestWaitForTurn(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		td := newDirection(New(Config{}), kindUpload)
		td.activeWorkers = 1

		results := make(chan bool, 2)

		go func() { results <- td.waitForTurn(1) }()

		synctest.Wait()
		require.Empty(t, results, "worker 1 must wait while one worker is active")

		td.workerMu.Lock()
		td.activeWorkers = 2
		td.workerMu.Unlock()
		td.workerCond.Broadcast()

		assert.True(t, <-results, "worker 1 resumes when two workers are active")

		go func() { results <- td.waitForTurn(5) }()

		synctest.Wait()
		require.Empty(t, results)

		td.stopped.Store(true)
		td.stopWorkers()

		assert.False(t, <-results, "a waiting worker gives up when the test stops")
	})
}

// TestRunUploadWorkers checks that uploads start at most as many concurrent workers as CPUs within the
// upload cap, while downloads run every worker. The CPU count is fixed at 2, below the cap, so the test shows the
// gating on any machine.
func TestRunUploadWorkers(t *testing.T) {
	t.Parallel()

	const threads = 16

	tests := []struct {
		name     string
		testKind kind
		wantMax  int
		wantPeak int
	}{
		{
			name:     "upload",
			testKind: kindUpload,
			wantMax:  defaultUploadMaxWorkers,
			wantPeak: 2,
		},
		{name: "download", testKind: kindDownload, wantMax: threads, wantPeak: threads},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				dm := New(
					Config{
						Logger:               nil,
						Connections:          threads,
						CaptureTime:          100 * time.Millisecond,
						RateCaptureFrequency: 0,
					},
				)
				dm.uploadMaxWorkers = defaultUploadMaxWorkers
				dm.cpus = 2

				var (
					mu            sync.Mutex
					running, peak int
				)

				td := newDirection(dm, tt.testKind)
				td.addHandler(func() {
					mu.Lock()
					running++
					peak = max(peak, running)
					mu.Unlock()

					time.Sleep(10 * time.Millisecond)

					mu.Lock()
					running--
					mu.Unlock()
				}, threads)

				_, cancel := context.WithCancel(context.Background())
				td.Run(context.Background(), cancel, 0)

				assert.Equal(t, tt.wantMax, td.maxWorkers)
				assert.Equal(t, tt.wantPeak, peak)
			})
		})
	}
}

// TestWorkerLimits checks that a connection count sets the download workers and caps the upload workers, and that
// the defaults apply without one.
func TestWorkerLimits(t *testing.T) {
	t.Parallel()

	downloads, uploads := workerLimits(3)
	assert.Equal(t, 3, downloads)
	assert.Equal(t, 3, uploads)

	downloads, uploads = workerLimits(0)
	assert.Equal(t, runtime.NumCPU(), downloads)
	assert.Equal(t, defaultUploadMaxWorkers, uploads)
}

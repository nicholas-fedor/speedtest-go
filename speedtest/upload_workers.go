// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"math"
	"time"
)

// Upload worker tuning.
const (
	// defaultUploadMaxWorkers caps upload workers when no thread count is set. Each worker holds a request body in
	// the transport, so more workers than the uplink can drain only queue data that may never be confirmed.
	defaultUploadMaxWorkers = 8
	// uploadAdjustInterval is how often the upload worker count is reconsidered.
	uploadAdjustInterval = time.Second
	// uploadGrowthThreshold is the factor by which the confirmed rate must beat the best rate to add a worker.
	uploadGrowthThreshold = 1.05
	// uploadDropThreshold is the share of the best rate below which workers are removed.
	uploadDropThreshold = 0.85
	// uploadBacklogPerWorker is the unconfirmed upload volume each active worker may hold before workers are removed.
	uploadBacklogPerWorker = 2 * 1024 * 1024
)

// nextUploadWorkers decides the active upload worker count for the next interval.
//
// A confirmed rate more than 5% above the best so far adds a worker. A rate below 85% of the best, or a backlog
// above 2 MiB per active worker, removes workers in proportion to the shortfall, and the current rate becomes the
// new best to measure against. The count stays between 1 and maxWorkers.
//
// Parameters:
//   - active: the current active worker count.
//   - maxWorkers: the upload worker cap.
//   - rate: the confirmed upload rate over the last interval, in bytes per second.
//   - bestRate: the best confirmed rate so far, or 0 before the first measurement.
//   - backlog: the bytes read into the transport but not yet confirmed.
//
// Returns:
//   - int: the new active worker count.
//   - float64: the new best rate.
func nextUploadWorkers(
	active, maxWorkers int,
	rate, bestRate float64,
	backlog int64,
) (int, float64) {
	if bestRate == 0 || rate > bestRate*uploadGrowthThreshold {
		if active < maxWorkers {
			active++
		}

		return active, rate
	}

	backlogLimit := int64(active) * uploadBacklogPerWorker

	if active <= 1 || (rate >= bestRate*uploadDropThreshold && backlog <= backlogLimit) {
		return active, bestRate
	}

	factor := 1.0
	if rate < bestRate*uploadDropThreshold {
		factor = math.Min(factor, rate/bestRate)
	}

	if backlog > backlogLimit {
		factor = math.Min(factor, float64(backlogLimit)/float64(backlog))
	}

	target := int(math.Ceil(float64(active) * factor))
	if target >= active {
		target = active - 1
	}

	return max(target, 1), rate
}

// adaptUploadWorkers adjusts the active upload worker count once per interval from the confirmed rate and the
// backlog, until stop is closed. Intervals before the first confirmation are skipped. A later interval without new
// confirmations can still remove workers for a large backlog, but never for its missing rate.
//
// Parameters:
//   - stop: closed when the test ends.
func (td *TestDirection) adaptUploadWorkers(stop <-chan struct{}) {
	ticker := time.NewTicker(uploadAdjustInterval)
	defer ticker.Stop()

	previousConfirmed := td.GetTotalDataVolume()
	previousAt := time.Now()

	var bestRate float64

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			confirmed := td.GetTotalDataVolume()
			delta := confirmed - previousConfirmed
			elapsed := now.Sub(previousAt).Seconds()
			previousConfirmed = confirmed
			previousAt = now

			if elapsed <= 0 || (delta <= 0 && bestRate == 0) {
				continue
			}

			// Confirmations arrive one whole request at a time, so on a slow uplink an interval can confirm nothing
			// while uploads progress. Such an interval keeps the best rate and is judged on the backlog alone.
			rate := bestRate
			if delta > 0 {
				rate = float64(delta) / elapsed
			}

			backlog := max(td.GetTotalReadVolume()-confirmed, 0)

			td.workerMu.Lock()
			active, newBest := nextUploadWorkers(
				td.activeWorkers,
				td.maxWorkers,
				rate,
				bestRate,
				backlog,
			)
			changed := active != td.activeWorkers
			td.activeWorkers = active
			td.workerMu.Unlock()

			bestRate = newBest

			if changed {
				td.workerCond.Broadcast()
				dbg.Printf("Upload workers: %d/%d (confirmed: %.2f MB/s, backlog: %.2f MB)\n",
					active, td.maxWorkers, rate/1e6, float64(backlog)/1e6)
			}
		}
	}
}

// waitForTurn blocks a worker until its number is within the active worker count or the test stops.
//
// Parameters:
//   - worker: the worker's number, counted from 0.
//
// Returns:
//   - bool: true when the worker should send a request, or false when the test has stopped.
func (td *TestDirection) waitForTurn(worker int) bool {
	td.workerMu.Lock()
	defer td.workerMu.Unlock()

	for {
		if !td.manager.isRunning() {
			return false
		}

		if worker < td.activeWorkers {
			return true
		}

		td.workerCond.Wait()
	}
}

// stopWorkers wakes every waiting worker so it sees that the test has stopped. Broadcasting under the lock means
// a worker that is checking the running state either sees the stop or is already waiting for this wake-up.
func (td *TestDirection) stopWorkers() {
	td.workerMu.Lock()
	defer td.workerMu.Unlock()

	td.workerCond.Broadcast()
}

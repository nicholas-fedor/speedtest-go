// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"math"
	"runtime"
	"time"
)

// Upload worker tuning.
const (
	// defaultUploadMaxWorkers caps upload workers when no connection count is set. Each worker holds a request body in
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

// workerLimits returns the worker counts for a configured number of connections.
//
// Parameters:
//   - connections: the configured connections, or less than 1 for the defaults.
//
// Returns:
//   - downloadWorkers: the download workers, one per CPU by default.
//   - uploadMaxWorkers: the most upload workers, 8 by default, since more workers than the uplink can drain only
//     queue unconfirmed data.
//
//nolint:nonamedreturns // Same-type returns need names.
func workerLimits(connections int) (downloadWorkers, uploadMaxWorkers int) {
	if connections < 1 {
		return runtime.NumCPU(), defaultUploadMaxWorkers
	}

	return connections, connections
}

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
//   - ctx: the test's context, for debug messages.
//   - stop: closed when the test ends.
func (d *Direction) adaptUploadWorkers(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(uploadAdjustInterval)
	defer ticker.Stop()

	previousConfirmed := d.confirmed.Load()
	previousAt := time.Now()

	var bestRate float64

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			confirmed := d.confirmed.Load()
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

			backlog := max(d.read.Load()-confirmed, 0)

			d.workerMu.Lock()
			active, newBest := nextUploadWorkers(
				d.activeWorkers,
				d.maxWorkers,
				rate,
				bestRate,
				backlog,
			)
			changed := active != d.activeWorkers
			d.activeWorkers = active
			d.workerMu.Unlock()

			bestRate = newBest

			if changed {
				d.workerCond.Broadcast()
				d.meter.logger.Debug(
					ctx,
					"upload workers",
					logKeyWorkers, active,
					logKeyLimit, d.maxWorkers,
					logKeyRate, rate,
					logKeyBacklog, backlog,
				)
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
func (d *Direction) waitForTurn(worker int) bool {
	d.workerMu.Lock()
	defer d.workerMu.Unlock()

	for {
		if d.stopped.Load() {
			return false
		}

		if worker < d.activeWorkers {
			return true
		}

		d.workerCond.Wait()
	}
}

// stopWorkers wakes every waiting worker so it sees that the test has stopped. Broadcasting under the lock means
// a worker that is checking the stop flag either sees the stop or is already waiting for this wake-up.
func (d *Direction) stopWorkers() {
	d.workerMu.Lock()
	defer d.workerMu.Unlock()

	d.workerCond.Broadcast()
}

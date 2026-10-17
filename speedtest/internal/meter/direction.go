// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/welford"
)

// kind is the direction of a test.
type kind int

// Direction runs the request handlers of one download or upload test and measures its rate.
type Direction struct {
	// stoppedAt is when the stop function ran.
	stoppedAt time.Time
	// meter is the meter the direction belongs to.
	meter *Meter
	// welford estimates the live rate and decides when it has settled. welfordMu guards it.
	welford *welford.Welford
	// closeFunc stops the test. It is set when the test starts.
	closeFunc func()
	// workerCond wakes workers when the active count changes or the test stops.
	workerCond *sync.Cond
	// handlers are the registered request handlers. handlersMu guards them.
	handlers []Handler
	// connections are the workers' connections, set when the test starts.
	connections []*Connection
	// rateSequence is the history of non-zero volumes per sample. welfordMu guards it.
	rateSequence []int64
	// stopBytes are the connections' bytes when the test stopped, in connection order.
	stopBytes []int64
	// result is the measurement, taken by the stop function before the workers are told to stop, and completed with
	// the request counts once they have.
	result Result
	// kind is whether the direction downloads or uploads.
	kind kind
	// activeWorkers is how many workers may send requests, numbered from 0. workerMu guards it.
	activeWorkers int
	// maxWorkers is how many workers were started. workerMu guards it.
	maxWorkers int
	// read counts the upload bytes read into the transport, confirmed or not.
	read atomic.Int64
	// handlersMu guards handlers.
	handlersMu sync.Mutex
	// welfordMu guards welford and rateSequence.
	welfordMu sync.Mutex
	// workerMu guards activeWorkers and maxWorkers.
	workerMu sync.Mutex
	// stopped reports that the direction's test has ended, so its workers and chunks stop transferring.
	stopped atomic.Bool
}

// Rate sampling.
const (
	// welfordWindowSize is the window the stability check looks back over.
	welfordWindowSize = 5 * time.Second
	// mainLoadFactor is the share of a multi-server test's handlers that run on the main server's workers.
	mainLoadFactor = 0.1
)

// Directions of a test.
const (
	// kindDownload measures downloads.
	kindDownload kind = iota
	// kindUpload measures uploads.
	kindUpload
)

// newDirection returns a direction with no handlers.
//
// Parameters:
//   - meter: the meter it belongs to.
//   - testKind: whether it downloads or uploads.
//
// Returns:
//   - *Direction: the direction.
func newDirection(meter *Meter, testKind kind) *Direction {
	direction := &Direction{meter: meter, kind: testKind}
	direction.workerCond = sync.NewCond(&direction.workerMu)

	return direction
}

// FinalRate returns the rate measured over the test, once [Direction.Run] has returned.
//
// Returns:
//   - float64: the bytes per second over the capture window.
func (d *Direction) FinalRate() float64 {
	return d.result.Rate
}

// Result returns the measurement of the test, once [Direction.Run] has returned.
//
// Returns:
//   - Result: the rate, bytes, and request counts, in total and for each connection.
func (d *Direction) Result() Result {
	return d.result
}

// Run runs the registered handlers on the direction's workers and blocks until the test stops.
//
// The test stops when the capture time ends, when the rate settles, or when ctx ends, so the workers do not keep
// retrying failed requests. Downloads run one worker per connection. Uploads start with no more workers than CPUs
// and let the adaptive worker count grow or shrink. In a multi-server test, the main handler gets a tenth of the
// workers and the others share the rest.
//
// Parameters:
//   - ctx: stops the test when it ends.
//   - cancel: cancels the requests' context when the test stops.
//   - mainHandler: the index of the handler that gets the main share of the workers.
func (d *Direction) Run(ctx context.Context, cancel context.CancelFunc, mainHandler int) {
	handlers := d.snapshotHandlers()
	if len(handlers) == 0 {
		panic("meter: no request handlers registered")
	}

	if mainHandler > len(handlers)-1 {
		mainHandler = 0
	}

	workerLimit := d.meter.connections
	if d.kind == kindUpload {
		workerLimit = min(workerLimit, d.meter.uploadMaxWorkers)
	}

	mainWorkers := max(int(mainLoadFactor*float64(len(handlers))), 1)
	if len(handlers) == 1 {
		mainWorkers = workerLimit
	}

	otherWorkers := workerLimit - mainWorkers

	initialWorkers := workerLimit
	if d.kind == kindUpload {
		initialWorkers = min(workerLimit, d.meter.cpus)
	}

	d.workerMu.Lock()
	d.maxWorkers = workerLimit
	d.activeWorkers = initialWorkers
	d.workerMu.Unlock()

	d.connections = assignConnections(d, len(handlers), mainHandler, mainWorkers, otherWorkers)

	d.meter.logger.Debug(
		ctx,
		"starting transfer workers",
		logKeyHandlers, len(handlers),
		logKeyMainWorkers, mainWorkers,
		logKeyOtherWorkers, otherWorkers,
	)

	var once sync.Once

	stopAdapting := make(chan struct{})
	stopCapture := make(chan struct{})
	captureStart := time.Now()

	d.closeFunc = func() {
		once.Do(func() {
			// Take the measurement at the moment the test stops, before requests are cancelled, so neither bytes
			// that arrive while they shut down nor a slow rate callback affect it.
			d.takeResult(captureStart)

			close(stopCapture)
			close(stopAdapting)
			d.stopped.Store(true)
			d.stopWorkers()
			cancel()
			d.meter.logger.Debug(ctx, "stopped transfer workers")
		})
	}

	d.rateCapture(captureStart, stopCapture)

	stopWatching := context.AfterFunc(ctx, d.closeFunc)
	defer stopWatching()

	if d.kind == kindUpload && workerLimit > 1 {
		d.meter.logger.Debug(
			ctx,
			"upload workers",
			logKeyWorkers,
			initialWorkers,
			logKeyLimit,
			workerLimit,
		)

		go d.adaptUploadWorkers(ctx, stopAdapting)
	}

	time.AfterFunc(d.meter.captureTime, d.closeFunc)

	var workers sync.WaitGroup

	for worker, connection := range d.connections {
		handler := handlers[connection.handler]

		workers.Go(func() {
			for d.waitForTurn(worker) {
				connection.run(handler)
			}
		})
	}

	workers.Wait()
	d.completeResult()
}

// assignConnections returns a connection for each worker of a test. The main handler's workers come first, and the
// other workers cycle through the remaining handlers in order.
//
// Parameters:
//   - direction: the test.
//   - handlers: the number of registered handlers.
//   - mainHandler: the index of the main handler.
//   - mainWorkers: the workers that run the main handler.
//   - otherWorkers: the workers that share the other handlers.
//
// Returns:
//   - []*Connection: the connections, in worker order.
func assignConnections(
	direction *Direction,
	handlers, mainHandler, mainWorkers, otherWorkers int,
) []*Connection {
	connections := make([]*Connection, 0, mainWorkers+max(otherWorkers, 0))

	for range mainWorkers {
		connections = append(connections, newConnection(direction, mainHandler))
	}

	for assigned := 0; assigned < otherWorkers; {
		for handler := range handlers {
			if assigned == otherWorkers {
				break
			}

			if handler == mainHandler {
				continue
			}

			connections = append(connections, newConnection(direction, handler))
			assigned++
		}
	}

	return connections
}

// addHandler adds a request handler, unless the direction already has limit of them.
//
// Parameters:
//   - handler: sends one request.
//   - limit: the most handlers the direction holds.
func (d *Direction) addHandler(handler Handler, limit int) {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	if len(d.handlers) < limit {
		d.handlers = append(d.handlers, handler)
	}
}

// snapshotHandlers returns a copy of the registered handlers.
//
// Returns:
//   - []Handler: the handlers.
func (d *Direction) snapshotHandlers() []Handler {
	d.handlersMu.Lock()
	defer d.handlersMu.Unlock()

	return append([]Handler{}, d.handlers...)
}

// volume returns the bytes the test's connections have counted so far.
//
// Returns:
//   - int64: the bytes downloaded, or the upload bytes the server confirmed.
func (d *Direction) volume() int64 {
	var total int64

	for _, connection := range d.connections {
		total += connection.Bytes()
	}

	return total
}

// takeResult records the rate and bytes at the moment the test stops.
//
// Parameters:
//   - start: when the capture window began.
func (d *Direction) takeResult(start time.Time) {
	d.stoppedAt = time.Now()
	d.stopBytes = make([]int64, len(d.connections))

	var total int64

	for index, connection := range d.connections {
		d.stopBytes[index] = connection.Bytes()
		total += d.stopBytes[index]
	}

	window := d.stoppedAt.Sub(start)
	d.result = Result{
		Connections: nil,
		Rate:        measuredRate(total, window),
		Bytes:       total,
		Sent:        d.read.Load(),
		Window:      window,
		Requests:    0,
		Failures:    0,
	}
}

// completeResult adds the request counts and each connection's measurement to the result, once every worker has
// returned.
func (d *Direction) completeResult() {
	d.result.Connections = make([]ConnectionResult, 0, len(d.connections))

	for index, connection := range d.connections {
		result := connection.result(d.stopBytes[index], d.stoppedAt)
		d.result.Connections = append(d.result.Connections, result)
		d.result.Requests += result.Requests
		d.result.Failures += result.Failures
	}
}

// rateCapture samples the transferred volume at each capture interval, updating the live rate and calling the rate
// callback, until stop is closed.
//
// Parameters:
//   - start: when the capture window began.
//   - stop: closed to end the capture.
func (d *Direction) rateCapture(start time.Time, stop <-chan struct{}) {
	frequency := d.meter.rateCaptureFrequency
	ticker := time.NewTicker(frequency)

	var previousVolume int64

	d.welfordMu.Lock()
	d.welford = welford.NewWelford(welfordWindowSize, frequency)
	d.welfordMu.Unlock()

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				volume := d.volume()
				delta := volume - previousVolume
				previousVolume = volume

				d.welfordMu.Lock()
				if delta != 0 {
					d.rateSequence = append(d.rateSequence, delta)
				}

				globalAvg := measuredRate(volume, time.Since(start))
				shouldStop := d.welford.Update(
					globalAvg,
					rateSample(d.kind, globalAvg, delta, frequency),
				)
				ewma := d.welford.EWMA()
				untilStable, settling := d.welford.UpdatesUntilStable()
				d.welfordMu.Unlock()

				if shouldStop && d.closeFunc != nil {
					go d.closeFunc()
				}

				if d.meter.onProgress != nil {
					elapsed := time.Since(start)

					d.meter.onProgress(Progress{
						Rate:    ewma,
						Elapsed: elapsed,
						Remaining: remaining(
							d.meter.captureTime,
							elapsed,
							untilStable,
							settling,
							frequency,
						),
					})
				}
			case <-stop:
				return
			}
		}
	}()
}

// remaining estimates the time left in a test: until the capture time ends, or until the stability check is
// expected to stop the test when that comes first.
//
// Parameters:
//   - captureTime: the longest the test runs.
//   - elapsed: the time since the test started measuring.
//   - untilStable: the rate captures the stability check still needs, assuming each is stable.
//   - settling: whether the rate is settling, so the stability check's estimate can hold.
//   - frequency: the time between rate captures.
//
// Returns:
//   - time.Duration: the estimated time left, never negative.
func remaining(
	captureTime, elapsed time.Duration,
	untilStable int,
	settling bool,
	frequency time.Duration,
) time.Duration {
	left := max(captureTime-elapsed, 0)
	if settling {
		left = min(left, time.Duration(untilStable)*frequency)
	}

	return left
}

// measuredRate returns the average rate of a volume transferred over a duration.
//
// Parameters:
//   - volume: the bytes transferred.
//   - elapsed: the time they took.
//
// Returns:
//   - float64: the rate in bytes per second, or 0 when no time has passed.
func measuredRate(volume int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}

	return float64(volume) / elapsed.Seconds()
}

// rateSample returns the value a rate capture period feeds to the EWMA estimator.
//
// Downloads count bytes as they arrive, so the period's own volume is a fair sample. Uploads count a whole request
// once the server confirms it, so period volumes alternate between zero and a full request. For uploads the sample
// is the cumulative average instead, scaled to one period, which avoids reporting that sawtooth.
//
// Parameters:
//   - testKind: whether the test downloads or uploads.
//   - globalAvg: the cumulative rate since capture started, in bytes per second.
//   - delta: the bytes counted during this period.
//   - frequency: the capture period.
//
// Returns:
//   - float64: the bytes to record for this period.
func rateSample(testKind kind, globalAvg float64, delta int64, frequency time.Duration) float64 {
	if testKind == kindUpload {
		return globalAvg * frequency.Seconds()
	}

	return float64(delta)
}

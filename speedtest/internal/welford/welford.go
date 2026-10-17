// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package welford

import (
	"fmt"
	"math"
	"time"
)

// Welford tracks the mean and standard deviation of a moving window of rate samples, with Welford's online
// algorithm, and an estimate of the rate that blends an exponentially weighted moving average with the window's
// average.
//
// See Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products.
// Technometrics, 4(3), 419–420. https://doi.org/10.1080/00401706.1962.10490022
type Welford struct {
	// vector holds the window's cumulative averages, used as a ring once it is full.
	vector []float64
	// movingVector holds the window's scaled samples, in the same slots as vector.
	movingVector []float64
	// n is the number of samples in the window, up to cap.
	n int
	// cap is the window size in samples.
	cap int
	// mean is the mean of the cumulative averages in the window.
	mean float64
	// sum is the sum of squared differences from the mean, from which the variance follows.
	sum float64
	// eraseIndex is the slot of the oldest sample, replaced by the next one once the window is full.
	eraseIndex int
	// currentStdDev is the standard deviation of the window's cumulative averages.
	currentStdDev float64
	// consecutiveStableIterations counts updates with a full window and a coefficient of variation below 3%, less
	// one for each update without.
	consecutiveStableIterations int
	// consecutiveStableIterationsThreshold is the count of stable updates that marks the series as stable.
	consecutiveStableIterationsThreshold int
	// cv is the coefficient of variation of the window's cumulative averages.
	cv float64
	// ewmaMean is the exponentially weighted moving average of the scaled samples.
	ewmaMean float64
	// steps is the number of updates so far.
	steps int
	// minSteps is the number of updates before the series can be reported stable.
	minSteps int
	// beta is the smoothing factor of the moving average.
	beta float64
	// scale converts a sample per sampling period to a rate per second.
	scale float64
	// movingAvg is the sum of the scaled samples in the window.
	movingAvg float64
}

// Window tuning.
const (
	// stabilityThresholdDivisor sets the stable updates needed to a third of the window.
	stabilityThresholdDivisor = 3
	// minSamplesForVariance is the number of samples a variance needs.
	minSamplesForVariance = 2
	// ewmaBetaNumerator gives the moving average the smoothing factor 2/(N+1) of an N-sample window.
	ewmaBetaNumerator = 2
	// minWindowSize is the smallest window, since a variance needs two samples.
	minWindowSize = 2
	// stableCV is the coefficient of variation below which an update of a full window counts as stable.
	stableCV = 0.03
)

// NewWelford returns an estimator whose window spans one cycle of samples taken at the given frequency, and at
// least two samples.
//
// The series counts as stable only after twice the window's samples, and after a third of the window's updates in
// a row, net of unstable ones, have seen a coefficient of variation below 3%.
//
// Parameters:
//   - cycle: the time the window covers.
//   - frequency: the time between samples.
//
// Returns:
//   - *Welford: the estimator.
func NewWelford(cycle, frequency time.Duration) *Welford {
	windowSize := max(int(cycle/frequency), minWindowSize)

	return &Welford{
		vector:                               make([]float64, windowSize),
		movingVector:                         make([]float64, windowSize),
		cap:                                  windowSize,
		consecutiveStableIterationsThreshold: windowSize / stabilityThresholdDivisor,
		minSteps:                             windowSize * 2,
		beta:                                 ewmaBetaNumerator / (float64(windowSize) + 1),
		scale:                                float64(time.Second / frequency),
	}
}

// Update adds a sample and reports whether the series has become stable.
//
// Parameters:
//   - globalAvg: the cumulative average rate since the test began, whose spread decides stability.
//   - value: the bytes counted in the last sampling period, which feeds the rate estimate.
//
// Returns:
//   - bool: true once the series is stable.
func (w *Welford) Update(globalAvg, value float64) bool {
	value *= w.scale
	if w.n == w.cap {
		delta := w.vector[w.eraseIndex] - w.mean
		w.mean -= delta / float64(w.n-1)
		w.sum -= delta * (w.vector[w.eraseIndex] - w.mean)
		// Removing a sample can leave a tiny negative rounding error.
		if w.sum < 0 {
			w.sum = 0
		}

		w.vector[w.eraseIndex] = globalAvg
		w.movingAvg -= w.movingVector[w.eraseIndex]
		w.movingVector[w.eraseIndex] = value
		w.movingAvg += value

		w.eraseIndex++
		if w.eraseIndex == w.cap {
			w.eraseIndex = 0
		}
	} else {
		w.vector[w.n] = globalAvg
		w.movingVector[w.n] = value
		w.movingAvg += value
		w.n++
	}

	delta := globalAvg - w.mean
	w.mean += delta / float64(w.n)
	w.sum += delta * (globalAvg - w.mean)
	w.currentStdDev = math.Sqrt(w.Variance())

	if w.mean != 0 {
		w.cv = w.currentStdDev / w.mean
	}

	w.ewmaMean = value*w.beta + w.ewmaMean*(1-w.beta)

	if w.n == w.cap && w.cv < stableCV {
		w.consecutiveStableIterations++
	} else if w.consecutiveStableIterations > 0 {
		w.consecutiveStableIterations--
	}

	w.steps++

	return w.consecutiveStableIterations >= w.consecutiveStableIterationsThreshold &&
		w.steps > w.minSteps
}

// UpdatesUntilStable estimates how many more updates the series needs before [Welford.Update] reports it stable,
// assuming each of them is stable. The window must still fill, a third of it must then be stable in a row, and the
// series must pass its minimum number of updates.
//
// The estimate is plausible while the series is settling, and also before its minimum number of updates, since a
// rate that is still ramping up usually settles by then. A series past its minimum that still swings may take any
// time to settle.
//
// Returns:
//   - int: the estimated number of updates, 0 once the series is stable.
//   - bool: true when the estimate is plausible.
func (w *Welford) UpdatesUntilStable() (int, bool) {
	settling := w.n >= minSamplesForVariance && w.cv < stableCV
	if !settling && w.steps > w.minSteps {
		return 0, false
	}

	toFill := w.cap - w.n
	toStable := toFill + max(
		w.consecutiveStableIterationsThreshold-w.consecutiveStableIterations,
		0,
	)
	toMinimum := w.minSteps + 1 - w.steps

	return max(toStable, toMinimum, 0), true
}

// Mean returns the mean of the window's cumulative averages.
//
// Returns:
//   - float64: the mean.
func (w *Welford) Mean() float64 {
	return w.mean
}

// CV returns the coefficient of variation of the window's cumulative averages.
//
// Returns:
//   - float64: the standard deviation divided by the mean, or 0 while the mean is 0.
func (w *Welford) CV() float64 {
	return w.cv
}

// Variance returns the sample variance of the window's cumulative averages.
//
// Returns:
//   - float64: the variance, or 0 with fewer than two samples.
func (w *Welford) Variance() float64 {
	if w.n < minSamplesForVariance {
		return 0
	}

	return w.sum / float64(w.n-1)
}

// StandardDeviation returns the standard deviation of the window's cumulative averages as of the last update.
//
// Returns:
//   - float64: the standard deviation.
func (w *Welford) StandardDeviation() float64 {
	return w.currentStdDev
}

// EWMA returns the rate estimate: the mean of the exponentially weighted moving average and the window's average
// of the scaled samples.
//
// Returns:
//   - float64: the rate in bytes per second.
func (w *Welford) EWMA() float64 {
	return w.ewmaMean*0.5 + w.movingAvg/float64(w.n)*0.5
}

// String describes the estimator's statistics.
//
// Returns:
//   - string: the mean, standard deviation, coefficient of variation, and rate estimate.
func (w *Welford) String() string {
	return fmt.Sprintf("Mean: %.2f, Standard Deviation: %.2f, C.V: %.2f, EWMA: %.2f",
		w.Mean(), w.StandardDeviation(), w.CV(), w.EWMA())
}

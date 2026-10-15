package welford

import (
	"testing"
	"time"
)

// BenchmarkWelford_Update measures an update with a constant sample for windows of 10, 100, and 1000 samples.
func BenchmarkWelford_Update(b *testing.B) {
	benchmarks := []struct {
		name       string
		windowSize int
	}{
		{"WindowSize10", 10},
		{"WindowSize100", 100},
		{"WindowSize1000", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cycle := time.Duration(bm.windowSize) * 100 * time.Millisecond
			frequency := 100 * time.Millisecond
			welford := NewWelford(cycle, frequency)

			globalAvg := 100.0
			value := 50.0

			b.ResetTimer()

			for b.Loop() {
				welford.Update(globalAvg, value)
			}
		})
	}
}

// BenchmarkWelford_UpdateWithVariedValues measures updates with samples that change on every call.
func BenchmarkWelford_UpdateWithVariedValues(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)

	values := []float64{50.0, 60.0, 40.0, 70.0, 30.0}
	globalAvgs := []float64{100.0, 105.0, 95.0, 110.0, 90.0}

	for i := 0; b.Loop(); i++ {
		idx := i % len(values)
		welford.Update(globalAvgs[idx], values[idx])
	}
}

// BenchmarkWelford_Mean measures reading the mean of a filled window.
func BenchmarkWelford_Mean(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)
	// Fill the window
	for range 10 {
		welford.Update(100.0, 50.0)
	}

	for b.Loop() {
		_ = welford.Mean()
	}
}

// BenchmarkWelford_Variance measures computing the variance of a filled window.
func BenchmarkWelford_Variance(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)
	// Fill the window
	for range 10 {
		welford.Update(100.0, 50.0)
	}

	for b.Loop() {
		_ = welford.Variance()
	}
}

// BenchmarkWelford_StandardDeviation measures reading the standard deviation of a filled window.
func BenchmarkWelford_StandardDeviation(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)
	// Fill the window
	for range 10 {
		welford.Update(100.0, 50.0)
	}

	for b.Loop() {
		_ = welford.StandardDeviation()
	}
}

// BenchmarkWelford_CV measures reading the coefficient of variation of a filled window.
func BenchmarkWelford_CV(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)
	// Fill the window
	for range 10 {
		welford.Update(100.0, 50.0)
	}

	for b.Loop() {
		_ = welford.CV()
	}
}

// BenchmarkWelford_EWMA measures computing the rate estimate of a filled window.
func BenchmarkWelford_EWMA(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)
	// Fill the window
	for range 10 {
		welford.Update(100.0, 50.0)
	}

	for b.Loop() {
		_ = welford.EWMA()
	}
}

package internal

import (
	"testing"
	"time"
)

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

func BenchmarkWelford_UpdateWithVariedValues(b *testing.B) {
	welford := NewWelford(time.Second, 100*time.Millisecond)

	values := []float64{50.0, 60.0, 40.0, 70.0, 30.0}
	globalAvgs := []float64{100.0, 105.0, 95.0, 110.0, 90.0}

	for i := 0; b.Loop(); i++ {
		idx := i % len(values)
		welford.Update(globalAvgs[idx], values[idx])
	}
}

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

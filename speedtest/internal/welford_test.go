package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWelford(t *testing.T) {
	type args struct {
		cycle     time.Duration
		frequency time.Duration
	}

	tests := []struct {
		name                     string
		args                     args
		wantCap                  int
		wantMinSteps             int
		wantConsecutiveThreshold int
		wantBeta                 float64
		wantScale                float64
	}{
		{
			name:                     "basic case",
			args:                     args{cycle: time.Second, frequency: 100 * time.Millisecond},
			wantCap:                  10,
			wantMinSteps:             20,
			wantConsecutiveThreshold: 3,
			wantBeta:                 2.0 / 11.0,
			wantScale:                10.0,
		},
		{
			name: "different durations",
			args: args{
				cycle:     2 * time.Second,
				frequency: 200 * time.Millisecond,
			},
			wantCap:                  10,
			wantMinSteps:             20,
			wantConsecutiveThreshold: 3,
			wantBeta:                 2.0 / 11.0,
			wantScale:                5.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewWelford(tt.args.cycle, tt.args.frequency)
			assert.NotNil(t, got)
			assert.Equal(t, tt.wantCap, got.cap)
			assert.Len(t, got.vector, tt.wantCap)
			assert.Len(t, got.movingVector, tt.wantCap)
			assert.Equal(t, tt.wantMinSteps, got.minSteps)
			assert.Equal(t, tt.wantConsecutiveThreshold, got.consecutiveStableIterationsThreshold)
			assert.InDelta(t, tt.wantBeta, got.beta, 1e-9)
			assert.InDelta(t, tt.wantScale, got.scale, 1e-9)
		})
	}
}

func TestWelford_Update(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	type args struct {
		globalAvg float64
		value     float64
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "initial update not stable",
			fields: fields{
				n:                                    0,
				cap:                                  3,
				vector:                               make([]float64, 3),
				movingVector:                         make([]float64, 3),
				consecutiveStableIterationsThreshold: 1,
				minSteps:                             5,
				beta:                                 0.5,
				scale:                                1.0,
			},
			args: args{globalAvg: 10.0, value: 1.0},
			want: false,
		},
		{
			name: "stable condition met",
			fields: fields{
				n:                                    3,
				cap:                                  3,
				vector:                               []float64{10.0, 10.0, 10.0},
				mean:                                 10.0,
				sum:                                  0.0,
				movingVector:                         []float64{1.0, 1.0, 1.0},
				movingAvg:                            3.0,
				consecutiveStableIterations:          2,
				consecutiveStableIterationsThreshold: 1,
				steps:                                6,
				minSteps:                             5,
				beta:                                 0.5,
				scale:                                1.0,
			},
			args: args{globalAvg: 10.0, value: 1.0},
			want: true,
		},
		{
			name: "not stable due to high cv",
			fields: fields{
				n:                                    3,
				cap:                                  3,
				vector:                               []float64{10.0, 10.0, 10.0},
				mean:                                 10.0,
				sum:                                  1.0,
				movingVector:                         []float64{1.0, 1.0, 1.0},
				movingAvg:                            3.0,
				consecutiveStableIterations:          0,
				consecutiveStableIterationsThreshold: 1,
				steps:                                6,
				minSteps:                             5,
				beta:                                 0.5,
				scale:                                1.0,
			},
			args: args{globalAvg: 10.0, value: 1.0},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:   tt.fields.n,
				cap: tt.fields.cap,
				vector: append(
					[]float64(nil),
					tt.fields.vector...), // copy slice
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector: append(
					[]float64(nil),
					tt.fields.movingVector...), // copy slice
				movingAvg: tt.fields.movingAvg,
			}
			got := w.Update(tt.args.globalAvg, tt.args.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWelford_Mean(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{"positive mean", fields{mean: 5.0}, 5.0},
		{"zero mean", fields{mean: 0.0}, 0.0},
		{"negative mean", fields{mean: -2.5}, -2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.InDelta(t, tt.want, w.Mean(), 1e-9)
		})
	}
}

func TestWelford_CV(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{"coefficient of variation", fields{cv: 0.1}, 0.1},
		{"zero cv", fields{cv: 0.0}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.InDelta(t, tt.want, w.CV(), 1e-9)
		})
	}
}

func TestWelford_Variance(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{"n less than 2", fields{n: 1, sum: 10.0}, 0.0},
		{"n equals 2", fields{n: 2, sum: 4.0}, 4.0},
		{"n greater than 2", fields{n: 3, sum: 6.0}, 3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.InDelta(t, tt.want, w.Variance(), 1e-9)
		})
	}
}

func TestWelford_StandardDeviation(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{"positive std dev", fields{currentStdDev: 2.5}, 2.5},
		{"zero std dev", fields{currentStdDev: 0.0}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.InDelta(t, tt.want, w.StandardDeviation(), 1e-9)
		})
	}
}

func TestWelford_EWMA(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{"ewma calculation", fields{ewmaMean: 4.0, movingAvg: 6.0, n: 2}, 3.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.InDelta(t, tt.want, w.EWMA(), 1e-9)
		})
	}
}

func TestWelford_String(t *testing.T) {
	type fields struct {
		n                                    int
		cap                                  int
		vector                               []float64
		mean                                 float64
		sum                                  float64
		eraseIndex                           int
		currentStdDev                        float64
		consecutiveStableIterations          int
		consecutiveStableIterationsThreshold int
		cv                                   float64
		ewmaMean                             float64
		steps                                int
		minSteps                             int
		beta                                 float64
		scale                                float64
		movingVector                         []float64
		movingAvg                            float64
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"string representation",
			fields{mean: 10.5, currentStdDev: 2.1, cv: 0.2, ewmaMean: 9.8, movingAvg: 9.8, n: 1},
			"Mean: 10.50, Standard Deviation: 2.10, C.V: 0.20, EWMA: 9.80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &Welford{
				n:                                    tt.fields.n,
				cap:                                  tt.fields.cap,
				vector:                               tt.fields.vector,
				mean:                                 tt.fields.mean,
				sum:                                  tt.fields.sum,
				eraseIndex:                           tt.fields.eraseIndex,
				currentStdDev:                        tt.fields.currentStdDev,
				consecutiveStableIterations:          tt.fields.consecutiveStableIterations,
				consecutiveStableIterationsThreshold: tt.fields.consecutiveStableIterationsThreshold,
				cv:                                   tt.fields.cv,
				ewmaMean:                             tt.fields.ewmaMean,
				steps:                                tt.fields.steps,
				minSteps:                             tt.fields.minSteps,
				beta:                                 tt.fields.beta,
				scale:                                tt.fields.scale,
				movingVector:                         tt.fields.movingVector,
				movingAvg:                            tt.fields.movingAvg,
			}
			assert.Equal(t, tt.want, w.String())
		})
	}
}

package welford

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// measuredRates is the live rate of a real download, sampled every 50 ms, in the order the rate capture fed it to
// Update.
var measuredRates = []float64{
	0, 6.91552, 18.721692307692308, 23.04116556291391,
	28.059485148514852, 31.118470119521916, 34.21727152317881,
	35.15127065527066, 36.74378054862843, 38.05981374722838,
	39.122272, 38.76271506352087, 39.60149084858569,
	40.23165538461539, 40.72287589158345, 41.3182940397351,
	41.36879800498754, 41.848093896713614, 42.29560044395117,
	42.45441684210526, 42.27466666666667, 42.67902476190476,
	42.68457142857143, 42.80731190269331, 42.75244,
	42.6367104, 42.65657801691007, 42.68948038490007,
	42.81048394004282, 42.64527084769124, 42.802427145708585,
	42.92410838709678, 43.15598501872659, 43.32482909090909,
	43.46838800705467, 43.5166464877213, 43.523846751804555,
	43.73983171521036, 43.870409258285115, 43.84913230769231,
	43.72752023988006, 43.85322926829268, 43.964127436994765,
	44.027410506741056, 43.99742169768498, 44.093007552199026,
	44.241321164710996, 44.287108464483204, 44.079746772178254,
	44.180878367346935, 44.1128768, 44.17403607843137,
	44.0556123076923, 44.20335873255375, 44.19607703703704,
	44.2603155216285, 44.202986438258385, 44.2371857042747,
	44.17902344827586, 44.31277559322034, 44.30876912840985,
	44.378359108781126, 44.3999845410628, 44.37440913415794,
	44.4760624609619, 44.471889264841586, 44.454861296184134,
	44.37803098927294, 44.37345251396648, 44.47449043478261,
	44.43488914285714, 44.50349070422535, 44.45223882254929,
	44.433060821917806, 44.46777081081081, 44.4554752,
	44.43440842105263, 44.55939356178609, 44.59433376057421,
	44.633178734177214, 44.5508462884279, 44.58072493827161,
	44.67904608632041, 44.662959537572256, 44.66403237324446,
	44.59772449459332, 44.53588837209303, 44.586143382352944,
	44.66949738933031, 44.66949618320611, 44.69251965356429,
	44.628363956043955, 44.63443946968051, 44.607556129032254,
	44.66488851063829, 44.66553062513155, 44.72077,
	44.75405813234384, 44.821391020408164, 44.88321874368814,
	44.9364192, 44.94112373787369, 44.958842352941176,
	44.97548797517455, 44.94797846153846, 44.9455939047619,
	44.910117647058826, 44.88617040358744, 44.86545696835092,
	44.86558503026968, 44.90363345454546, 44.87501062871554,
	44.93104802713802, 44.93590804597701, 44.90068409051044,
	44.87467130434782, 44.891225650749874, 44.9156841025641,
	44.92164610169491, 44.89092118971601, 44.9027482086319,
	44.86017701453105, 44.87692428711898, 44.8651103526735,
	44.8483070967742, 44.79754023356263, 44.787605776860815,
	44.7891175822446, 44.778797499999996, 44.778773283743995,
	44.79586406274028, 44.77517923664122, 44.62143350499849,
	44.48119933904161, 44.38141850746269, 44.425831753554505,
	44.390321423320096, 44.31049810218978, 44.124642318840586,
	44.01107798561151, 44.02091885714285, 44.02881950942861,
	44.019682253521125, 44.071147412587415, 44.08087753401833,
	44.09411310344827, 44.12980821917809, 44.07751870493811,
	44.09083445854713, 44.09876506104924, 44.08840634582056,
	44.13032264900663, 44.09793658729115, 44.106114509803916,
	44.06832567199065, 44.096165139981935, 44.116,
	44.10581480066234, 44.09257851448817, 44.075851106639846,
	44.074916, 44.091910073282826, 44.09682567901235,
	44.07930895705522, 44.053896892138944, 44.06240019391589,
	44.06116150788872, 44.1180531736527, 44.15990285714285,
	44.134685126020585, 44.18049829431832, 44.158170252572496,
	44.21358883720931, 44.16577849710982, 44.1291108045977,
	44.16022671694664, 44.1493200045439, 44.1491597740113,
	44.15741842696629, 44.177169162011175, 44.21226797022553,
	44.166602585349686, 44.17985497692815, 44.168419672131144,
	44.19028366481904, 44.173513455095645, 44.162090322580646,
	44.123481608212145, 44.155531914893615, 44.18765714285714,
	44.224294736842104, 44.21350366492147, 44.168821324448146,
	44.168, 44.17472356215214, 44.187647999999996,
	44.18084571428571, 44.18716962744899, 44.19440137359862,
	44.1429404079992, 44.15704,
}

// TestNewWelford checks the window size, stability threshold, minimum updates, smoothing factor, and scale derived
// from the cycle and sampling frequency.
func TestNewWelford(t *testing.T) {
	t.Parallel()

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

// TestNewWelford_MinimumWindow checks that a cycle shorter than two samples still gives a two-sample window, so
// updates neither index an empty window nor divide by a single sample's n-1.
func TestNewWelford_MinimumWindow(t *testing.T) {
	t.Parallel()

	for _, cycle := range []time.Duration{0, 10 * time.Millisecond, 50 * time.Millisecond} {
		welford := NewWelford(cycle, 50*time.Millisecond)

		assert.Equal(t, minWindowSize, welford.cap)
		require.NotPanics(t, func() {
			for _, rate := range []float64{100, 110, 105, 108} {
				welford.Update(rate, rate)
			}
		})
		assert.False(t, math.IsNaN(welford.Mean()), "cycle %s", cycle)
		assert.False(t, math.IsNaN(welford.EWMA()), "cycle %s", cycle)
	}
}

// TestWelford_Update checks the stability decision: a first update is never stable, a full window with a low
// coefficient of variation is, and a high coefficient of variation is not.
func TestWelford_Update(t *testing.T) {
	t.Parallel()

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
					tt.fields.vector...,
				), // copy slice
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
					tt.fields.movingVector...,
				), // copy slice
				movingAvg: tt.fields.movingAvg,
			}
			got := w.Update(tt.args.globalAvg, tt.args.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWelford_Mean checks that Mean reports the tracked mean, whatever its sign.
func TestWelford_Mean(t *testing.T) {
	t.Parallel()

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

// TestWelford_CV checks that CV reports the tracked coefficient of variation.
func TestWelford_CV(t *testing.T) {
	t.Parallel()

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

// TestWelford_Variance checks the sample variance, which needs at least two samples.
func TestWelford_Variance(t *testing.T) {
	t.Parallel()

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

// TestWelford_StandardDeviation checks that StandardDeviation reports the value from the last update.
func TestWelford_StandardDeviation(t *testing.T) {
	t.Parallel()

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

// TestWelford_EWMA checks that the rate estimate averages the moving average with the window's mean sample.
func TestWelford_EWMA(t *testing.T) {
	t.Parallel()

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

// TestWelford_String checks the description of the estimator's statistics.
func TestWelford_String(t *testing.T) {
	t.Parallel()

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

// TestWelford_UpdateConverges checks that a real rate series, which rises quickly and then levels off, is reported
// stable only once it has levelled off, at its last sample. A change to the stability rule moves that sample.
func TestWelford_UpdateConverges(t *testing.T) {
	t.Parallel()

	welford := NewWelford(5*time.Second, 50*time.Millisecond)

	stableAt := slices.IndexFunc(
		measuredRates,
		func(rate float64) bool { return welford.Update(rate, rate) },
	)

	require.NotEqual(t, -1, stableAt, "the series never became stable")
	assert.Equal(t, len(measuredRates)-1, stableAt)
}

// TestWelford_UpdatesUntilStable checks the estimate against the real rate series: it never predicts a later stop
// than the actual one, since it assumes every coming update is stable, it predicts the actual stop exactly once the
// series has levelled off, and it reaches 0 at the stop.
func TestWelford_UpdatesUntilStable(t *testing.T) {
	t.Parallel()

	welford := NewWelford(5*time.Second, 50*time.Millisecond)
	stop := len(measuredRates) - 1

	var predictions []int

	for index, rate := range measuredRates {
		welford.Update(rate, rate)

		if remaining, ok := welford.UpdatesUntilStable(); ok {
			assert.LessOrEqual(t, index+remaining, stop, "update %d predicts a later stop", index)
			predictions = append(predictions, index+remaining)
		}
	}

	require.GreaterOrEqual(t, len(predictions), 50)

	for _, predicted := range predictions[len(predictions)-50:] {
		assert.Equal(t, stop, predicted)
	}

	remaining, ok := welford.UpdatesUntilStable()
	assert.True(t, ok)
	assert.Zero(t, remaining)
}

// TestWelford_UpdatesUntilStable_Unsettled checks a series that keeps swinging: before its minimum number of updates
// the estimate assumes it settles by then, and past the minimum it gives no estimate.
func TestWelford_UpdatesUntilStable_Unsettled(t *testing.T) {
	t.Parallel()

	// A 1-second window of 50-millisecond samples has 20 samples, so its minimum is 40 updates.
	welford := NewWelford(time.Second, 50*time.Millisecond)

	welford.Update(10, 10)

	remaining, ok := welford.UpdatesUntilStable()
	assert.True(t, ok, "a series that just started is expected to settle by its minimum")
	assert.Equal(t, 40, remaining)

	for index := range 39 {
		rate := float64(10 + 20*(index%2))
		welford.Update(rate, rate)
	}

	remaining, ok = welford.UpdatesUntilStable()
	assert.True(t, ok, "a swinging series before its minimum is still expected to settle")
	assert.Equal(t, 6, remaining, "the window must still be stable for a third of it, rounded down")

	welford.Update(10, 10)
	welford.Update(30, 30)

	_, ok = welford.UpdatesUntilStable()
	assert.False(t, ok, "a swinging series past its minimum may take any time to settle")
}

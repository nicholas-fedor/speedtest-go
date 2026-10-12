package speedtest

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_funcGroup_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fn       func()
		initial  int
		expected int
	}{
		{
			name:     "add one function",
			fn:       func() {},
			initial:  0,
			expected: 1,
		},
		{
			name: "add another function",
			fn: func() {
				// do nothing
			},
			initial:  1,
			expected: 2,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			f := &funcGroup{fns: make([]func(), testCase.initial)}
			f.Add(testCase.fn)
			assert.Len(t, f.fns, testCase.expected)
		})
	}
}

func TestDataManager_NewDataDirection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testType int
	}{
		{
			name:     "create download direction",
			testType: typeDownload,
		},
		{
			name:     "create upload direction",
			testType: typeUpload,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.NewDataDirection(testCase.testType)
			assert.NotNil(t, got)
			assert.Equal(t, testCase.testType, got.TestType)
			assert.NotNil(t, got.manager)
			assert.NotNil(t, got.funcGroup)
		})
	}
}

func TestNewDataManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *DataManager
	}{
		{
			name: "create new data manager",
			want: &DataManager{
				nThread:              runtime.NumCPU(),
				captureTime:          time.Second * 15,
				rateCaptureFrequency: time.Millisecond * 50,
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewDataManager()
			assert.NotNil(t, got)
			assert.Equal(t, testCase.want.nThread, got.nThread)
			assert.Equal(t, testCase.want.captureTime, got.captureTime)
			assert.Equal(t, testCase.want.rateCaptureFrequency, got.rateCaptureFrequency)
			assert.NotNil(t, got.download)
			assert.NotNil(t, got.upload)
			assert.NotNil(t, got.SnapshotStore)
		})
	}
}

func TestDataManager_SetCallbackDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "set callback download",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			callback := func(_ ByteRate) {}
			dm.SetCallbackDownload(callback)
			// Test passes if no panic
		})
	}
}

func TestDataManager_SetCallbackUpload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "set callback upload",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			callback := func(_ ByteRate) {}
			dm.SetCallbackUpload(callback)
			// Test passes if no panic
		})
	}
}

func TestDataManager_Wait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "wait for data manager",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dm.Wait() // Should return immediately since no active tasks
		})
	}
}

func TestDataManager_RegisterUploadHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "register upload handler",
			fn:   func() {},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.RegisterUploadHandler(testCase.fn)
			assert.Equal(t, dm.upload, got)
			assert.Len(t, got.fns, 1)
		})
	}
}

func TestDataManager_RegisterDownloadHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "register download handler",
			fn:   func() {},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.RegisterDownloadHandler(testCase.fn)
			assert.Equal(t, dm.download, got)
			assert.Len(t, got.fns, 1)
		})
	}
}

func TestTestDirection_GetTotalDataVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int64
	}{
		{
			name: "initial volume",
			want: 0,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			td := dm.NewDataDirection(typeDownload)
			got := td.GetTotalDataVolume()
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestTestDirection_AddTotalDataVolume(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		delta int64
		want  int64
	}{
		{
			name:  "add positive delta",
			delta: 100,
			want:  100,
		},
		{
			name:  "add negative delta",
			delta: -50,
			want:  -50,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			td := dm.NewDataDirection(typeDownload)
			got := td.AddTotalDataVolume(testCase.delta)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestTestDirection_Start(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "start test direction",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			testDirection := dm.NewDataDirection(typeDownload)
			testDirection.Add(func() {}) // Add a function to avoid panic

			_, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()

			testDirection.Start(cancel, 0) // Should start and be cancelled
		})
	}
}

func TestTestDirection_rateCapture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "rate capture",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The bubble fails the test if the capture goroutine is still running when it ends.
			synctest.Test(t, func(_ *testing.T) {
				td := NewDataManager().NewDataDirection(typeDownload)
				stop := make(chan struct{})
				td.rateCapture(time.Now(), stop)

				time.Sleep(3 * td.manager.rateCaptureFrequency)
				close(stop)
			})
		})
	}
}

func TestDataManager_NewChunk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "new chunk",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.NewChunk()
			assert.NotNil(t, got)
			assert.Equal(t, dm, got.GetParent())
		})
	}
}

func TestDataManager_AddTotalDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int64
		want  int64
	}{
		{
			name:  "add download value",
			value: 1000,
			want:  1000,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dm.AddTotalDownload(testCase.value)
			got := dm.GetTotalDownload()
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestDataManager_AddTotalUpload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int64
		want  int64
	}{
		{
			name:  "add upload value",
			value: 2000,
			want:  2000,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dm.AddTotalUpload(testCase.value)
			got := dm.GetTotalUpload()
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestDataManager_GetTotalDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int64
	}{
		{
			name: "initial download",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetTotalDownload()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDataManager_GetTotalUpload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int64
	}{
		{
			name: "initial upload",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetTotalUpload()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDataManager_SetRateCaptureFrequency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
	}{
		{
			name:     "set rate capture frequency",
			duration: time.Millisecond * 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.SetRateCaptureFrequency(tt.duration)
			assert.Equal(t, dm, got)
			assert.Equal(t, tt.duration, dm.rateCaptureFrequency)
		})
	}
}

func TestDataManager_SetCaptureTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
	}{
		{
			name:     "set capture time",
			duration: time.Second * 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.SetCaptureTime(tt.duration)
			assert.Equal(t, dm, got)
			assert.Equal(t, tt.duration, dm.captureTime)
		})
	}
}

func TestDataManager_SetNThread(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want int
	}{
		{
			name: "set n thread",
			n:    4,
			want: 4,
		},
		{
			name: "set n thread zero",
			n:    0,
			want: runtime.NumCPU(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.SetNThread(tt.n)
			assert.Equal(t, dm, got)
			assert.Equal(t, tt.want, dm.nThread)
		})
	}
}

func TestDataManager_Snapshots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "get snapshots",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.Snapshots()
			assert.Equal(t, dm.SnapshotStore, got)
			assert.NotNil(t, got)
		})
	}
}

func TestDataManager_Reset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "reset data manager",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			originalDownload := dm.download
			originalUpload := dm.upload
			dm.Reset()
			assert.NotSame(t, originalDownload, dm.download)
			assert.NotSame(t, originalUpload, dm.upload)
		})
	}
}

func TestDataManager_GetAvgDownloadRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want float64
	}{
		{
			name: "average download rate",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetAvgDownloadRate()
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestDataManager_GetEWMADownloadRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want float64
	}{
		{
			name: "EWMA download rate",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetEWMADownloadRate()
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestDataManager_GetAvgUploadRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want float64
	}{
		{
			name: "average upload rate",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetAvgUploadRate()
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestDataManager_GetEWMAUploadRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want float64
	}{
		{
			name: "EWMA upload rate",
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			got := dm.GetEWMAUploadRate()
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestDataChunk_GetDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want time.Duration
	}{
		{
			name: "get duration",
			want: time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dc := &DataChunk{
				startTime: time.Unix(0, 0),
				endTime:   time.Unix(1, 0),
			}
			got := dc.GetDuration()
			assert.Equal(t, time.Second, got)
		})
	}
}

// TestDataChunk_GetRate checks the download rate of a chunk. Fixed timestamps keep the span exactly one second, where
// two calls to time.Now drift apart under load.
func TestDataChunk_GetRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want float64
	}{
		{
			name: "get rate download",
			want: 1000.0, // 1000 bytes / 1 second
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dataChunk := &DataChunk{
				dateType:            typeDownload,
				startTime:           time.Unix(0, 0),
				endTime:             time.Unix(1, 0),
				remainOrDiscardSize: 1000,
			}
			got := dataChunk.GetRate()
			assert.InDelta(t, testCase.want, got, 0.001)
		})
	}
}

func TestDataChunk_DownloadHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "download handler",
			data:    "test data",
			wantErr: false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dc := dm.NewChunk()
			r := bytes.NewReader([]byte(testCase.data))

			err := dc.DownloadHandler(r)
			if testCase.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataChunk_UploadHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int64
	}{
		{
			name: "upload handler",
			size: 1000,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dc := dm.NewChunk()
			got := dc.UploadHandler(testCase.size)
			assert.Equal(t, dc, got)
		})
	}
}

func TestDataChunk_GetParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "get parent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dc := dm.NewChunk()
			got := dc.GetParent()
			assert.Equal(t, dm, got)
		})
	}
}

func TestDataChunk_Read(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		size    int64
		b       []byte
		wantN   int
		wantErr bool
	}{
		{
			name:    "read from chunk",
			size:    100,
			b:       make([]byte, 50),
			wantN:   50,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dm := NewDataManager()
			dc := dm.NewChunk()
			dc.UploadHandler(tt.size)

			gotN, err := dc.Read(tt.b)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantN, gotN)
		})
	}
}

func TestCalcMAFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		list []int64
		want float64
	}{
		{
			name: "calculate MA filter",
			list: []int64{1, 2, 3, 4, 5},
			want: 3.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := calcMAFilter(tt.list)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func Test_pautaFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vector []int64
		want   []int64
	}{
		{
			name:   "pauta filter",
			vector: []int64{100, 101, 102, 200},
			want:   []int64{100, 101, 102, 200},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pautaFilter(tt.vector)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_sampleVariance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		vector     []int64
		wantMean   int64
		wantStdDev int64
	}{
		{
			name:       "sample variance",
			vector:     []int64{1, 2, 3, 4, 5},
			wantMean:   3,
			wantStdDev: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMean, gotStdDev := sampleVariance(tt.vector)
			assert.Equal(t, tt.wantMean, gotMean)
			assert.Equal(t, tt.wantStdDev, gotStdDev)
		})
	}
}

func Test_newHistorySnapshots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
		want int
	}{
		{
			name: "new history snapshots",
			size: 5,
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newHistorySnapshots(tt.size)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want, got.maxSize)
		})
	}
}

func TestSnapshots_push(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		size  int
		value *Snapshot
		want  int
	}{
		{
			name:  "push snapshot",
			size:  5,
			value: &Snapshot{},
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rs := newHistorySnapshots(tt.size)
			rs.push(tt.value)
			assert.Len(t, rs.sp, tt.want)
		})
	}
}

func TestSnapshots_Latest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		size  int
		value *Snapshot
		want  *Snapshot
	}{
		{
			name:  "latest snapshot",
			size:  5,
			value: &Snapshot{},
			want:  &Snapshot{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rs := newHistorySnapshots(tt.size)
			rs.push(tt.value)
			got := rs.Latest()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSnapshots_All(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		size  int
		value *Snapshot
		want  int
	}{
		{
			name:  "all snapshots",
			size:  5,
			value: &Snapshot{},
			want:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rs := newHistorySnapshots(tt.size)
			rs.push(tt.value)
			got := rs.All()
			assert.Len(t, got, tt.want)
		})
	}
}

func TestSnapshots_Clean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
	}{
		{
			name: "clean snapshots",
			size: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rs := newHistorySnapshots(tt.size)
			rs.push(&Snapshot{})
			rs.Clean()
			assert.Empty(t, rs.sp)
		})
	}
}

func TestFuncGroup_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	f := &funcGroup{fns: make([]func(), 0)}

	var wg sync.WaitGroup

	numGoroutines := 100

	for i := range numGoroutines {
		wg.Add(1)

		go func(val int) {
			defer wg.Done()

			f.Add(func() { _ = val })
		}(i)
	}

	wg.Wait()
	assert.Len(t, f.snapshotFns(), numGoroutines)
}

func TestDataManager_ConcurrentReset(t *testing.T) {
	t.Parallel()

	dm := NewDataManager()

	var wg sync.WaitGroup

	for i := range 100 {
		wg.Go(func() {
			dm.Reset()
		})

		if i%10 == 0 {
			wg.Go(func() {
				_ = dm.GetTotalDownload()
				_ = dm.GetTotalUpload()
			})
		}
	}

	wg.Wait()
}

func TestDataManager_ConcurrentHandlers(t *testing.T) {
	t.Parallel()

	dm := NewDataManager()

	var wg sync.WaitGroup

	numHandlers := 50

	for range numHandlers {
		wg.Go(func() {
			dm.RegisterDownloadHandler(func() {})
			dm.RegisterUploadHandler(func() {})
		})
	}

	wg.Wait()
}

// TestDataChunk_ReadDoesNotCountUpload checks that reading an upload chunk leaves the upload total alone. The
// transport reads ahead into buffers, so bytes are counted only once the server confirms the request.
func TestDataChunk_ReadDoesNotCountUpload(t *testing.T) {
	t.Parallel()

	dm := NewDataManager()
	chunk := dm.NewChunk().UploadHandler(3 * readChunkSize)

	read, err := io.Copy(io.Discard, chunk)
	require.NoError(t, err)

	assert.Equal(t, int64(3*readChunkSize), read)
	assert.Zero(t, dm.GetTotalUpload())
}

// TestRateSample checks that downloads sample the period's own volume and uploads sample the cumulative average
// scaled to one period.
func TestRateSample(t *testing.T) {
	t.Parallel()

	const frequency = 50 * time.Millisecond

	tests := []struct {
		name      string
		testType  int
		globalAvg float64
		delta     int64
		want      float64
	}{
		{
			name:      "download uses the delta",
			testType:  typeDownload,
			globalAvg: 2000,
			delta:     300,
			want:      300,
		},
		{name: "download idle period", testType: typeDownload, globalAvg: 2000, delta: 0, want: 0},
		{
			name:      "upload uses the average",
			testType:  typeUpload,
			globalAvg: 2000,
			delta:     300,
			want:      100,
		},
		{name: "upload idle period", testType: typeUpload, globalAvg: 2000, delta: 0, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.InDelta(
				t,
				tt.want,
				rateSample(tt.testType, tt.globalAvg, tt.delta, frequency),
				1e-9,
			)
		})
	}
}

// TestTestDirection_rateCaptureBursts feeds a whole chunk every other capture period, the pattern confirmed uploads
// produce, and checks the reported rate once it has settled over the first half of the periods. Uploads then rise
// steadily instead of dipping on the periods without new bytes. Downloads, which sample each period's own volume,
// still dip on every such period, which shows the pattern tells the two apart.
func TestTestDirection_rateCaptureBursts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testType int
		wantDips int
	}{
		{name: "upload", testType: typeUpload, wantDips: 0},
		{name: "download", testType: typeDownload, wantDips: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				dm := NewDataManager()
				td := dm.NewDataDirection(tt.testType)

				var (
					mu    sync.Mutex
					rates []ByteRate
				)

				td.captureCallback = func(rate ByteRate) {
					mu.Lock()
					defer mu.Unlock()

					rates = append(rates, rate)
				}

				stop := make(chan struct{})
				td.rateCapture(time.Now(), stop)

				// Bursts land halfway between ticks, so no burst races a tick.
				time.Sleep(dm.rateCaptureFrequency / 2)

				for period := range 20 {
					if period%2 == 0 {
						td.AddTotalDataVolume(64 * 1024)
					}

					time.Sleep(dm.rateCaptureFrequency)
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

// TestStartContextFinalRate checks that the final rate is the measured volume over the capture window. One
// megabyte arrives every 100 ms, ten times in a 1.05 s window, for both directions. The live estimate starts at zero
// and would still read well below this when the window ends.
func TestStartContextFinalRate(t *testing.T) {
	t.Parallel()

	for _, testType := range []int{typeDownload, typeUpload} {
		synctest.Test(t, func(t *testing.T) {
			dm := NewDataManager()
			dm.SetNThread(1)
			dm.captureTime = 1050 * time.Millisecond

			td := dm.NewDataDirection(testType)
			td.Add(func() {
				time.Sleep(100 * time.Millisecond)
				td.AddTotalDataVolume(1_000_000)
			})

			_, cancel := context.WithCancel(context.Background())
			td.StartContext(context.Background(), cancel, 0)

			assert.InDelta(t, 10_000_000/1.05, td.finalRate, 1e-6)

			// The capture can still be finishing a tick after the stop, so read the live estimate under its lock.
			td.welfordMu.Lock()
			ewma := td.welford.EWMA()
			td.welfordMu.Unlock()

			assert.Less(t, ewma, 0.9*td.finalRate, "the live estimate is still converging")
		})
	}
}

// TestStartContextFinalRateExcludesCancellation checks that bytes arriving while requests are cancelled, after the
// capture stops, are left out of the final rate.
func TestStartContextFinalRateExcludesCancellation(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		dm := NewDataManager()
		dm.SetNThread(1)
		dm.captureTime = 100 * time.Millisecond

		td := dm.NewDataDirection(typeDownload)

		var once sync.Once

		td.Add(func() {
			once.Do(func() { td.AddTotalDataVolume(1_000) })
			time.Sleep(10 * time.Millisecond)
		})

		td.StartContext(context.Background(), func() { td.AddTotalDataVolume(1_000_000) }, 0)

		assert.Equal(t, int64(1_001_000), td.GetTotalDataVolume())
		assert.InDelta(
			t,
			10_000.0,
			td.finalRate,
			1e-9,
			"only the 1,000 bytes before the stop count",
		)
	})
}

// TestStartContextFinalRateIgnoresBlockedCallback checks that a rate callback that blocks neither delays the stop nor
// changes the final rate: the test stops on time, and bytes that arrive while the capture is still busy are left
// out, whichever case the capture handles next.
func TestStartContextFinalRateIgnoresBlockedCallback(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		dm := NewDataManager()
		dm.SetNThread(1)
		dm.captureTime = 100 * time.Millisecond

		td := dm.NewDataDirection(typeDownload)
		release := make(chan struct{})

		td.captureCallback = func(ByteRate) { <-release }

		var once sync.Once

		td.Add(func() {
			once.Do(func() { td.AddTotalDataVolume(1_000) })
			time.Sleep(10 * time.Millisecond)
		})

		start := time.Now()

		_, cancel := context.WithCancel(context.Background())
		td.StartContext(context.Background(), cancel, 0)

		// The stop comes at 100 ms. A worker that started another 10 ms request at that same instant finishes it first.
		assert.LessOrEqual(
			t,
			time.Since(start),
			110*time.Millisecond,
			"a blocked callback must not delay the stop",
		)

		td.AddTotalDataVolume(1_000_000)
		close(release)

		assert.InDelta(
			t,
			10_000.0,
			td.finalRate,
			1e-9,
			"only the 1,000 bytes before the stop count",
		)
	})
}

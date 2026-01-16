package speedtest

import (
	"bytes"
	"context"
	"runtime"
	"testing"
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

			dm := NewDataManager()
			td := dm.NewDataDirection(typeDownload)
			got := td.rateCapture()
			assert.NotNil(t, got)
		})
	}
}

func TestDataManager_NewChunk(t *testing.T) {
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

func TestDataChunk_GetRate(t *testing.T) {
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
				startTime:           time.Now().Add(-time.Second),
				endTime:             time.Now(),
				remainOrDiscardSize: 1000,
			}
			got := dataChunk.GetRate()
			assert.InDelta(t, testCase.want, got, 0.001)
		})
	}
}

func TestDataChunk_DownloadHandler(t *testing.T) {
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

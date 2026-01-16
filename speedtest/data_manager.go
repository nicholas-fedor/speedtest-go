package speedtest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/showwin/speedtest-go/speedtest/internal"
)

const (
	defaultCaptureTime          = 15 * time.Second
	defaultRateCaptureFrequency = 50 * time.Millisecond
	welfordWindowSize           = 5 * time.Second
	conversionFactor            = 1000
	bitsToKbps                  = 1000
	blackHoleBufferSize         = 8192
	medianExclusionCount        = 2
	outlierThresholdFactor      = 3
)

// Manager defines the interface for managing data chunks and test directions.
type Manager interface {
	SetRateCaptureFrequency(duration time.Duration) Manager
	SetCaptureTime(duration time.Duration) Manager

	NewChunk() Chunk

	GetTotalDownload() int64
	GetTotalUpload() int64
	AddTotalDownload(value int64)
	AddTotalUpload(value int64)

	GetAvgDownloadRate() float64
	GetAvgUploadRate() float64

	GetEWMADownloadRate() float64
	GetEWMAUploadRate() float64

	SetCallbackDownload(callback func(downRate ByteRate))
	SetCallbackUpload(callback func(upRate ByteRate))

	RegisterDownloadHandler(fn func()) *TestDirection
	RegisterUploadHandler(fn func()) *TestDirection

	// Wait for the upload or download task to end to avoid errors caused by core occupation
	Wait()
	Reset()
	Snapshots() *Snapshots

	SetNThread(n int) Manager
}

// Chunk defines the interface for data chunks used in speed tests.
type Chunk interface {
	UploadHandler(size int64) Chunk
	DownloadHandler(r io.Reader) error

	GetRate() float64
	GetDuration() time.Duration
	GetParent() Manager

	Read(b []byte) (n int, err error)
}

const readChunkSize = 1024 // 1 KBytes with higher frequency rate feedback

// DataType represents the type of data chunk.
type DataType int32

const (
	typeEmptyChunk = iota
	typeDownload
	typeUpload
)

var (
	// ErrUninitializedManager is returned when the manager is not properly initialized.
	ErrUninitializedManager = errors.New("uninitialized manager")
	// ErrMultipleCallsChunkHandler is returned when multiple calls to the same chunk handler are not allowed.
	ErrMultipleCallsChunkHandler = errors.New(
		"multiple calls to the same chunk handler are not allowed",
	)
	// ErrFailedGetBufferPool is returned when failing to get buffer from pool.
	ErrFailedGetBufferPool = errors.New("failed to get buffer from pool")
)

type funcGroup struct {
	fns []func()
}

func (f *funcGroup) Add(fn func()) {
	f.fns = append(f.fns, fn)
}

// DataManager manages data chunks and test directions for speed tests.
type DataManager struct {
	sync.Mutex

	SnapshotStore *Snapshots
	Snapshot      *Snapshot

	repeatByte *[]byte

	captureTime          time.Duration
	rateCaptureFrequency time.Duration
	nThread              int

	running   bool
	runningRW sync.RWMutex

	download *TestDirection
	upload   *TestDirection
}

// TestDirection represents a direction of test (upload or download) with associated handlers.
type TestDirection struct {
	*funcGroup // actually exec function

	TestType        int                         // test type
	manager         *DataManager                // manager
	totalDataVolume int64                       // total send/receive data volume
	RateSequence    []int64                     // rate history sequence
	welford         *internal.Welford           // std/EWMA/mean
	captureCallback func(realTimeRate ByteRate) // user callback
	closeFunc       func()                      // close func
}

// NewDataManager creates a new DataManager instance with default settings.
func NewDataManager() *DataManager {
	repeatedData := bytes.Repeat(
		[]byte{0xAA},
		readChunkSize,
	) // uniformly distributed sequence of bits
	ret := &DataManager{
		nThread:              runtime.NumCPU(),
		captureTime:          defaultCaptureTime,
		rateCaptureFrequency: defaultRateCaptureFrequency,
		Snapshot:             &Snapshot{},
		repeatByte:           &repeatedData,
	}
	ret.download = ret.NewDataDirection(typeDownload)
	ret.upload = ret.NewDataDirection(typeUpload)
	ret.SnapshotStore = newHistorySnapshots(maxSnapshotSize)

	return ret
}

// NewDataDirection creates a new TestDirection for the specified test type.
func (dm *DataManager) NewDataDirection(testType int) *TestDirection {
	return &TestDirection{
		TestType:  testType,
		manager:   dm,
		funcGroup: &funcGroup{},
	}
}

// SetCallbackDownload sets the callback function for download rate updates.
func (dm *DataManager) SetCallbackDownload(callback func(downRate ByteRate)) {
	if dm.download != nil {
		dm.download.captureCallback = callback
	}
}

// SetCallbackUpload sets the callback function for upload rate updates.
func (dm *DataManager) SetCallbackUpload(callback func(upRate ByteRate)) {
	if dm.upload != nil {
		dm.upload.captureCallback = callback
	}
}

// Wait waits for the data manager to finish processing and returns when no more data is being transferred.
func (dm *DataManager) Wait() {
	oldDownTotal := dm.GetTotalDownload()

	oldUpTotal := dm.GetTotalUpload()
	for {
		time.Sleep(dm.rateCaptureFrequency)
		newDownTotal := dm.GetTotalDownload()
		newUpTotal := dm.GetTotalUpload()
		deltaDown := newDownTotal - oldDownTotal
		deltaUp := newUpTotal - oldUpTotal
		oldDownTotal = newDownTotal
		oldUpTotal = newUpTotal

		if deltaDown == 0 && deltaUp == 0 {
			return
		}
	}
}

// RegisterUploadHandler registers a handler function for upload operations.
func (dm *DataManager) RegisterUploadHandler(fn func()) *TestDirection {
	if len(dm.upload.fns) < dm.nThread {
		dm.upload.Add(fn)
	}

	return dm.upload
}

// RegisterDownloadHandler registers a handler function for download operations.
func (dm *DataManager) RegisterDownloadHandler(fn func()) *TestDirection {
	if len(dm.download.fns) < dm.nThread {
		dm.download.Add(fn)
	}

	return dm.download
}

// GetTotalDataVolume returns the total data volume transferred.
func (td *TestDirection) GetTotalDataVolume() int64 {
	return atomic.LoadInt64(&td.totalDataVolume)
}

// AddTotalDataVolume adds to the total data volume.
func (td *TestDirection) AddTotalDataVolume(delta int64) int64 {
	return atomic.AddInt64(&td.totalDataVolume, delta)
}

// Start begins the test direction execution with the given cancel function and main request handler index.
func (td *TestDirection) Start(cancel context.CancelFunc, mainRequestHandlerIndex int) {
	if len(td.fns) == 0 {
		panic("empty task stack")
	}

	if mainRequestHandlerIndex > len(td.fns)-1 {
		mainRequestHandlerIndex = 0
	}

	mainLoadFactor := 0.1
	// When the number of processor cores is equivalent to the processing program,
	// the processing efficiency reaches the highest level (VT is not considered).
	mainN := int(mainLoadFactor * float64(len(td.fns)))
	if mainN == 0 {
		mainN = 1
	}

	if len(td.fns) == 1 {
		mainN = td.manager.nThread
	}

	auxN := td.manager.nThread - mainN
	dbg.Printf("Available fns: %d\n", len(td.fns))
	dbg.Printf("mainN: %d\n", mainN)
	dbg.Printf("auxN: %d\n", auxN)

	waitGroup := sync.WaitGroup{}
	td.manager.running = true
	stopCapture := td.rateCapture()

	// refresh once function
	once := sync.Once{}
	td.closeFunc = func() {
		once.Do(func() {
			stopCapture <- true

			close(stopCapture)
			td.manager.runningRW.Lock()
			td.manager.running = false
			td.manager.runningRW.Unlock()
			cancel()
			dbg.Println("FuncGroup: Stop")
		})
	}

	time.AfterFunc(td.manager.captureTime, td.closeFunc)

	for range mainN {
		waitGroup.Go(func() {
			for {
				td.manager.runningRW.RLock()
				running := td.manager.running
				td.manager.runningRW.RUnlock()

				if !running {
					return
				}

				td.fns[mainRequestHandlerIndex]()
			}
		})
	}

	for auxIndex := 0; auxIndex < auxN; {
		for functionIndex := range td.fns {
			if auxIndex == auxN {
				break
			}

			if functionIndex == mainRequestHandlerIndex {
				continue
			}

			waitGroup.Add(1)

			taskIndex := functionIndex

			go func() {
				defer waitGroup.Done()

				for {
					td.manager.runningRW.RLock()
					running := td.manager.running
					td.manager.runningRW.RUnlock()

					if !running {
						return
					}

					td.fns[taskIndex]()
				}
			}()

			auxIndex++
		}
	}

	waitGroup.Wait()
}

func (td *TestDirection) rateCapture() chan bool {
	ticker := time.NewTicker(td.manager.rateCaptureFrequency)

	var prevTotalDataVolume int64

	stopCapture := make(chan bool)
	td.welford = internal.NewWelford(welfordWindowSize, td.manager.rateCaptureFrequency)
	sTime := time.Now()

	go func(t *time.Ticker) {
		defer t.Stop()

		for {
			select {
			case <-t.C:
				newTotalDataVolume := td.GetTotalDataVolume()
				deltaDataVolume := newTotalDataVolume - prevTotalDataVolume
				prevTotalDataVolume = newTotalDataVolume

				if deltaDataVolume != 0 {
					td.RateSequence = append(td.RateSequence, deltaDataVolume)
				}
				// anyway we update the measuring instrument
				globalAvg := (float64(td.GetTotalDataVolume())) / float64(
					time.Since(sTime).Milliseconds(),
				) * conversionFactor
				if td.welford.Update(globalAvg, float64(deltaDataVolume)) {
					go td.closeFunc()
				}
				// reports the current rate at the given rate
				if td.captureCallback != nil {
					td.captureCallback(ByteRate(td.welford.EWMA()))
				}
			case stop := <-stopCapture:
				if stop {
					return
				}
			}
		}
	}(ticker)

	return stopCapture
}

// NewChunk creates a new data chunk for the manager.
func (dm *DataManager) NewChunk() Chunk {
	var dataChunk DataChunk

	dataChunk.manager = dm
	dm.Lock()
	*dm.Snapshot = append(*dm.Snapshot, &dataChunk)
	dm.Unlock()

	return &dataChunk
}

// AddTotalDownload adds to the total download data volume.
func (dm *DataManager) AddTotalDownload(value int64) {
	dm.download.AddTotalDataVolume(value)
}

// AddTotalUpload adds to the total upload data volume.
func (dm *DataManager) AddTotalUpload(value int64) {
	dm.upload.AddTotalDataVolume(value)
}

// GetTotalDownload returns the total download data volume.
func (dm *DataManager) GetTotalDownload() int64 {
	return dm.download.GetTotalDataVolume()
}

// GetTotalUpload returns the total upload data volume.
func (dm *DataManager) GetTotalUpload() int64 {
	return dm.upload.GetTotalDataVolume()
}

// SetRateCaptureFrequency sets the frequency for capturing rate data.
func (dm *DataManager) SetRateCaptureFrequency(duration time.Duration) Manager {
	dm.rateCaptureFrequency = duration

	return dm
}

// SetCaptureTime sets the time duration for capturing data.
func (dm *DataManager) SetCaptureTime(duration time.Duration) Manager {
	dm.captureTime = duration

	return dm
}

// SetNThread sets the number of threads for the manager.
func (dm *DataManager) SetNThread(n int) Manager {
	if n < 1 {
		dm.nThread = runtime.NumCPU()
	} else {
		dm.nThread = n
	}

	return dm
}

// Snapshots returns the snapshots manager.
func (dm *DataManager) Snapshots() *Snapshots {
	return dm.SnapshotStore
}

// Reset resets the data manager to its initial state.
func (dm *DataManager) Reset() {
	dm.SnapshotStore.push(dm.Snapshot)
	dm.Snapshot = &Snapshot{}
	dm.download = dm.NewDataDirection(typeDownload)
	dm.upload = dm.NewDataDirection(typeUpload)
}

// GetAvgDownloadRate returns the average download rate.
// GetAvgDownloadRate returns the average download rate.
func (dm *DataManager) GetAvgDownloadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)

	return float64(dm.download.GetTotalDataVolume()*8/bitsToKbps) / unit
}

// GetEWMADownloadRate returns the exponentially weighted moving average download rate.
func (dm *DataManager) GetEWMADownloadRate() float64 {
	if dm.download.welford != nil {
		return dm.download.welford.EWMA()
	}

	return 0
}

// GetAvgUploadRate returns the average upload rate.
func (dm *DataManager) GetAvgUploadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)

	return float64(dm.upload.GetTotalDataVolume()*8/bitsToKbps) / unit
}

// GetEWMAUploadRate returns the exponentially weighted moving average upload rate.
func (dm *DataManager) GetEWMAUploadRate() float64 {
	if dm.upload.welford != nil {
		return dm.upload.welford.EWMA()
	}

	return 0
}

// DataChunk represents a chunk of data for speed tests.
type DataChunk struct {
	manager             *DataManager
	dateType            DataType
	startTime           time.Time
	endTime             time.Time
	err                 error
	ContentLength       int64
	remainOrDiscardSize int64
}

var blackHolePool = sync.Pool{
	New: func() any {
		b := make([]byte, blackHoleBufferSize)

		return &b
	},
}

// GetDuration returns the duration of the data chunk transfer.
func (dc *DataChunk) GetDuration() time.Duration {
	return dc.endTime.Sub(dc.startTime)
}

// GetRate returns the transfer rate of the data chunk.
func (dc *DataChunk) GetRate() float64 {
	switch dc.dateType {
	case typeDownload:
		return float64(dc.remainOrDiscardSize) / dc.GetDuration().Seconds()
	case typeUpload:
		return float64(
			dc.ContentLength-dc.remainOrDiscardSize,
		) * 8 / 1000 / 1000 / dc.GetDuration().
			Seconds()
	}

	return 0
}

// DownloadHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
func (dc *DataChunk) DownloadHandler(reader io.Reader) error {
	if dc.dateType != typeEmptyChunk {
		dc.err = ErrMultipleCallsChunkHandler

		return dc.err
	}

	dc.dateType = typeDownload
	dc.startTime = time.Now()

	defer func() {
		dc.endTime = time.Now()
	}()

	bufP, ok := blackHolePool.Get().(*[]byte)
	if !ok {
		return ErrFailedGetBufferPool
	}
	defer blackHolePool.Put(bufP)

	var readSize int

	for {
		dc.manager.runningRW.RLock()
		running := dc.manager.running
		dc.manager.runningRW.RUnlock()

		if !running {
			return nil
		}

		readSize, dc.err = reader.Read(*bufP)
		rs := int64(readSize)

		dc.remainOrDiscardSize += rs
		dc.manager.download.AddTotalDataVolume(rs)

		if dc.err != nil {
			if errors.Is(dc.err, io.EOF) {
				return nil
			}

			return dc.err
		}
	}
}

// UploadHandler initializes the data chunk for upload with the given size.
func (dc *DataChunk) UploadHandler(size int64) Chunk {
	if dc.dateType != typeEmptyChunk {
		dc.err = ErrMultipleCallsChunkHandler
	}

	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}

	dc.ContentLength = size
	dc.remainOrDiscardSize = size
	dc.dateType = typeUpload
	dc.startTime = time.Now()

	return dc
}

// GetParent returns the manager associated with this data chunk.
func (dc *DataChunk) GetParent() Manager {
	return dc.manager
}

func (dc *DataChunk) Read(buffer []byte) (int, error) {
	var bytesRead int

	if dc.remainOrDiscardSize < readChunkSize {
		if dc.remainOrDiscardSize <= 0 {
			dc.endTime = time.Now()

			return 0, io.EOF
		}

		bytesRead = copy(buffer, (*dc.manager.repeatByte)[:dc.remainOrDiscardSize])
	} else {
		bytesRead = copy(buffer, *dc.manager.repeatByte)
	}

	bytesRead64 := int64(bytesRead)
	dc.remainOrDiscardSize -= bytesRead64
	dc.manager.AddTotalUpload(bytesRead64)

	return bytesRead, nil
}

// calcMAFilter Median-Averaging Filter.
func calcMAFilter(list []int64) float64 {
	if len(list) == 0 {
		return 0
	}

	var sum int64

	listLength := len(list)
	if listLength == 0 {
		return 0
	}

	length := len(list)
	for i := range length - 1 {
		for j := i + 1; j < length; j++ {
			if list[i] > list[j] {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	for i := 1; i < listLength-1; i++ {
		sum += list[i]
	}

	return float64(sum) / float64(listLength-medianExclusionCount)
}

func pautaFilter(vector []int64) []int64 {
	dbg.Println("Per capture unit")
	dbg.Printf("Raw Sequence len: %d\n", len(vector))
	dbg.Printf("Raw Sequence: %v\n", vector)

	if len(vector) == 0 {
		return vector
	}

	mean, stdDev := sampleVariance(vector)

	var retVec []int64

	for _, value := range vector {
		if math.Abs(float64(value-mean)) < float64(outlierThresholdFactor*stdDev) {
			retVec = append(retVec, value)
		}
	}

	dbg.Printf("Raw average: %dByte\n", mean)
	dbg.Printf("Pauta Sequence len: %d\n", len(retVec))
	dbg.Printf("Pauta Sequence: %v\n", retVec)

	return retVec
}

// sampleVariance sample Variance.
func sampleVariance(vector []int64) (int64, int64) {
	if len(vector) == 0 {
		return 0, 0
	}

	var sumNum, accumulate, mean, stdDev int64

	for _, value := range vector {
		sumNum += value
	}

	mean = sumNum / int64(len(vector))
	for _, value := range vector {
		accumulate += (value - mean) * (value - mean)
	}

	variance := accumulate / int64(len(vector)-1) // Bessel's correction
	stdDev = int64(math.Sqrt(float64(variance)))

	return mean, stdDev
}

const maxSnapshotSize = 10

// Snapshot represents a snapshot of data chunks at a point in time.
type Snapshot []*DataChunk

// Snapshots manages a collection of snapshots.
type Snapshots struct {
	sp      []*Snapshot
	maxSize int
}

func newHistorySnapshots(size int) *Snapshots {
	return &Snapshots{
		sp:      make([]*Snapshot, 0, size),
		maxSize: size,
	}
}

// Latest returns the most recent snapshot.
func (rs *Snapshots) Latest() *Snapshot {
	if len(rs.sp) > 0 {
		return rs.sp[len(rs.sp)-1]
	}

	return nil
}

// All returns all snapshots.
func (rs *Snapshots) All() []*Snapshot {
	return rs.sp
}

// Clean clears all stored snapshots.
func (rs *Snapshots) Clean() {
	rs.sp = make([]*Snapshot, 0, rs.maxSize)
}

func (rs *Snapshots) push(value *Snapshot) {
	if len(rs.sp) == rs.maxSize {
		rs.sp = rs.sp[1:]
	}

	rs.sp = append(rs.sp, value)
}

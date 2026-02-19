package speedtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nicholas-fedor/speedtest-go/speedtest/transport"
)

type (
	downloadFunc func(context.Context, *Server, int) error
	uploadFunc   func(context.Context, *Server, int) error
	registerFunc func(func()) *TestDirection
	getRateFunc  func() float64
)

var (
	dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} // kB
)

// ErrConnectTimeout is returned when server connection times out.
var ErrConnectTimeout = errors.New("server connect timeout")

var (
	// ErrServerNil is returned when the server is nil.
	ErrServerNil = errors.New("server is nil")
	// ErrNoAvailableServers is returned when no available servers are found.
	ErrNoAvailableServers = errors.New("not found available servers")
)

func (s *Server) multiTestContext(
	ctx context.Context,
	servers Servers,
	register registerFunc,
	requestFunc func(context.Context, *Server, int) error,
	handlerName string,
	getRate getRateFunc,
	setSpeed func(ByteRate),
) error {
	if s == nil {
		return ErrServerNil
	}

	availableServers := servers.Available()
	if availableServers.Len() == 0 {
		return ErrNoAvailableServers
	}

	mainIDIndex := 0

	var testDirection *TestDirection

	_context, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		errorTimes   int64
		requestTimes int64
	)

	for i, availableServer := range *availableServers {
		if availableServer.ID == s.ID {
			mainIDIndex = i
		}

		server := availableServer
		dbg.Printf("Register %s Handler: %s\n", handlerName, server.URL)

		testDirection = register(func() {
			atomic.AddInt64(&requestTimes, 1)

			err := requestFunc(_context, server, 3)
			if err != nil {
				atomic.AddInt64(&errorTimes, 1)
			}
		})
	}

	if testDirection == nil {
		return ErrUninitializedManager
	}

	testDirection.Start(cancel, mainIDIndex) // block here

	rate := ByteRate(getRate())
	setSpeed(rate)

	if rate == 0 && float64(errorTimes)/float64(requestTimes) > 0.1 {
		setSpeed(-1) // N/A
	}

	return nil
}

// MultiDownloadTestContext executes multi-download test with context.
func (s *Server) MultiDownloadTestContext(ctx context.Context, servers Servers) error {
	if s == nil {
		return ErrServerNil
	}

	return s.multiTestContext(
		ctx,
		servers,
		s.Context.RegisterDownloadHandler,
		downloadRequest,
		"Download",
		s.Context.GetEWMADownloadRate,
		func(rate ByteRate) { s.DLSpeed = rate },
	)
}

// MultiUploadTestContext executes multi-upload test with context.
func (s *Server) MultiUploadTestContext(ctx context.Context, servers Servers) error {
	if s == nil {
		return ErrServerNil
	}

	return s.multiTestContext(
		ctx,
		servers,
		s.Context.RegisterUploadHandler,
		uploadRequest,
		"Upload",
		s.Context.GetEWMAUploadRate,
		func(rate ByteRate) { s.ULSpeed = rate },
	)
}

// DownloadTest executes the test to measure download speed.
func (s *Server) DownloadTest() error {
	if s == nil {
		return ErrServerNil
	}

	return s.downloadTestContext(context.Background(), downloadRequest)
}

// DownloadTestContext executes the test to measure download speed, observing the given context.
func (s *Server) DownloadTestContext(ctx context.Context) error {
	if s == nil {
		return ErrServerNil
	}

	return s.downloadTestContext(ctx, downloadRequest)
}

func (s *Server) testContext(
	ctx context.Context,
	requestFunc func(context.Context, *Server, int) error,
	size int,
	register registerFunc,
	getRate getRateFunc,
	setSpeed func(ByteRate),
	setDuration func(*time.Duration),
) error {
	if s == nil {
		return ErrServerNil
	}

	if s.Context == nil {
		return ErrUninitializedManager
	}

	var (
		errorTimes   int64
		requestTimes int64
	)

	start := time.Now()
	_context, cancel := context.WithCancel(ctx)
	register(func() {
		atomic.AddInt64(&requestTimes, 1)

		err := requestFunc(_context, s, size)
		if err != nil {
			atomic.AddInt64(&errorTimes, 1)
		}
	}).Start(cancel, 0)

	duration := time.Since(start)

	rate := ByteRate(getRate())
	if rate == 0 && float64(errorTimes)/float64(requestTimes) > 0.1 {
		rate = -1 // N/A
	}

	setSpeed(rate)
	setDuration(&duration)
	s.testDurationTotalCount()

	return nil
}

func (s *Server) downloadTestContext(ctx context.Context, downloadRequest downloadFunc) error {
	if s == nil {
		return ErrServerNil
	}

	return s.testContext(
		ctx,
		downloadRequest,
		3,
		s.Context.RegisterDownloadHandler,
		s.Context.GetEWMADownloadRate,
		func(rate ByteRate) { s.DLSpeed = rate },
		func(d *time.Duration) { s.TestDuration.Download = d },
	)
}

// UploadTest executes the test to measure upload speed.
func (s *Server) UploadTest() error {
	if s == nil {
		return ErrServerNil
	}

	return s.uploadTestContext(context.Background(), uploadRequest)
}

// UploadTestContext executes the test to measure upload speed, observing the given context.
func (s *Server) UploadTestContext(ctx context.Context) error {
	if s == nil {
		return ErrServerNil
	}

	return s.uploadTestContext(ctx, uploadRequest)
}

func (s *Server) uploadTestContext(ctx context.Context, uploadRequest uploadFunc) error {
	if s == nil {
		return ErrServerNil
	}

	return s.testContext(
		ctx,
		uploadRequest,
		4,
		s.Context.RegisterUploadHandler,
		s.Context.GetEWMAUploadRate,
		func(rate ByteRate) { s.ULSpeed = rate },
		func(d *time.Duration) { s.TestDuration.Upload = d },
	)
}

func downloadRequest(ctx context.Context, server *Server, writer int) error {
	if server == nil {
		return ErrServerNil
	}

	if server.Context == nil {
		return ErrUninitializedManager
	}

	size := dlSizes[writer]

	u, err := url.Parse(server.URL)
	if err != nil {
		return fmt.Errorf("failed to parse download URL: %w", err)
	}

	u.Path = path.Dir(u.Path)
	xdlURL := u.JoinPath(fmt.Sprintf("random%dx%d.jpg", size, size)).String()
	dbg.Printf("XdlURL: %s\n", xdlURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xdlURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download HTTP request: %w", err)
	}

	resp, err := server.Context.doer.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform download HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return fmt.Errorf(
		"failed to download data: %w",
		server.Context.NewChunk().DownloadHandler(resp.Body),
	)
}

func uploadRequest(ctx context.Context, server *Server, writer int) error {
	if server == nil {
		return ErrServerNil
	}

	if server.Context == nil {
		return ErrUninitializedManager
	}

	size := ulSizes[writer]
	chunkSize := int64(size*100-51) * 10
	dc := server.Context.NewChunk().UploadHandler(chunkSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, io.NopCloser(dc))
	if err != nil {
		return fmt.Errorf("failed to create upload HTTP request: %w", err)
	}

	req.ContentLength = chunkSize
	dbg.Printf("Len=%d, XulURL: %s\n", req.ContentLength, server.URL)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := server.Context.doer.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform upload HTTP request: %w", err)
	}

	_, _ = io.Copy(io.Discard, resp.Body)

	defer func() { _ = resp.Body.Close() }()

	if err != nil {
		return fmt.Errorf("failed to upload data: %w", err)
	}

	return nil
}

// PingTest executes test to measure latency.
func (s *Server) PingTest(callback func(latency time.Duration)) error {
	return s.PingTestContext(context.Background(), callback)
}

// PingTestContext executes test to measure latency, observing the given context.
func (s *Server) PingTestContext(ctx context.Context, callback func(latency time.Duration)) error {
	if s == nil {
		return ErrServerNil
	}

	if s.Context == nil {
		return ErrUninitializedManager
	}

	start := time.Now()

	var (
		vectorPingResult []int64
		err              error
	)

	switch s.Context.config.PingMode {
	case TCP:
		vectorPingResult, err = s.TCPPing(ctx, 10, time.Millisecond*200, callback)
	case ICMP:
		vectorPingResult, err = s.ICMPPing(ctx, time.Second*4, 10, time.Millisecond*200, callback)
	case HTTP:
		vectorPingResult, err = s.HTTPPing(ctx, 10, time.Millisecond*200, callback)
	default:
		vectorPingResult, err = s.HTTPPing(ctx, 10, time.Millisecond*200, callback)
	}

	if err != nil || len(vectorPingResult) == 0 {
		return err
	}

	dbg.Printf("Before StandardDeviation: %v\n", vectorPingResult)
	mean, _, std, minLatency, maxLatency := StandardDeviation(vectorPingResult)
	duration := time.Since(start)
	s.Latency = time.Duration(mean) * time.Nanosecond
	s.Jitter = time.Duration(std) * time.Nanosecond
	s.MinLatency = time.Duration(minLatency) * time.Nanosecond
	s.MaxLatency = time.Duration(maxLatency) * time.Nanosecond
	s.TestDuration.Ping = &duration
	s.testDurationTotalCount()

	return nil
}

// TestAll executes ping, download and upload tests one by one.
func (s *Server) TestAll() error {
	if s == nil {
		return ErrServerNil
	}

	err := s.PingTest(nil)
	if err != nil {
		return err
	}

	err = s.DownloadTest()
	if err != nil {
		return err
	}

	return s.UploadTest()
}

// TCPPing performs TCP ping test.
func (s *Server) TCPPing(
	ctx context.Context,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrServerNil
	}

	if s.Context == nil {
		return nil, ErrUninitializedManager
	}

	var pingDst string

	if len(s.Host) == 0 {
		u, err := url.Parse(s.URL)
		if err != nil || len(u.Host) == 0 {
			return nil, fmt.Errorf("failed to parse server URL for TCP ping: %w", err)
		}

		pingDst = u.Host
	} else {
		pingDst = s.Host
	}

	failTimes := 0

	latencies := make([]int64, 0, echoTimes)

	client, err := transport.NewClient(s.Context.tcpDialer)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport client for TCP ping: %w", err)
	}

	err = client.Connect(ctx, pingDst)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for TCP ping: %w", err)
	}

	for range echoTimes {
		latency, err := client.PingContext(ctx)
		if err != nil {
			failTimes++

			continue
		}

		latencies = append(latencies, latency)
		if callback != nil {
			callback(time.Duration(latency))
		}

		time.Sleep(echoFreq)
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	if err != nil {
		return latencies, fmt.Errorf("failed to perform TCP ping: %w", err)
	}

	return latencies, nil
}

// HTTPPing performs HTTP ping test.
func (s *Server) HTTPPing(
	ctx context.Context,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrServerNil
	}

	if s.Context == nil {
		return nil, ErrUninitializedManager
	}

	var contextErr error

	u, err := url.Parse(s.URL)
	if err != nil || len(u.Host) == 0 {
		return nil, fmt.Errorf("failed to parse server URL for TCP ping: %w", err)
	}

	u.Path = path.Dir(u.Path)
	pingDst := u.JoinPath("latency.txt").String()
	dbg.Printf("Echo: %s\n", pingDst)

	failTimes := 0
	latencies := make([]int64, 0, echoTimes+1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingDst, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping HTTP request: %w", err)
	}
	// carry out an extra request to warm up the connection and ensure the first request is not going to affect the
	// overall estimation
	echoTimes++
	for i := range echoTimes {
		sTime := time.Now()
		resp, err := s.Context.doer.Do(req)
		endTime := time.Since(sTime)

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				contextErr = err

				break
			}

			failTimes++

			continue
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if i > 0 {
			latency := endTime.Nanoseconds()
			latencies = append(latencies, latency)
			dbg.Printf("RTT: %d\n", latency)

			if callback != nil {
				callback(endTime)
			}
		}

		time.Sleep(echoFreq)
	}

	if contextErr != nil {
		return latencies, contextErr
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	if err != nil {
		return latencies, fmt.Errorf("failed to perform HTTP ping: %w", err)
	}

	return latencies, nil
}

// PingTimeout represents the timeout value for ping operations.
const (
	PingTimeout        = -1
	echoOptionDataSize = 32 // `echoMessage` need to change at same time
)

// ICMPPing privileged method.
func (s *Server) ICMPPing(
	ctx context.Context,
	readTimeout time.Duration,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrServerNil
	}

	if s.Context == nil {
		return nil, ErrUninitializedManager
	}

	latencies := make([]int64, 0, echoTimes)

	u, err := url.ParseRequestURI(s.URL)
	if err != nil || len(u.Host) == 0 {
		return nil, fmt.Errorf("failed to parse ICMP URL: %w", err)
	}

	dbg.Printf("Echo: %s\n", strings.Split(u.Host, ":")[0])

	dialContext, err := s.Context.ipDialer.DialContext(
		ctx,
		"ip:icmp",
		strings.Split(u.Host, ":")[0],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ICMP: %w", err)
	}

	defer func() { _ = dialContext.Close() }()

	icmpData := prepareICMPPacket()

	failTimes := 0

	for i := range echoTimes {
		latency, err := s.sendOneICMPPing(dialContext, icmpData, i, readTimeout)
		if err != nil {
			failTimes++

			continue
		}

		latencies = append(latencies, latency.Nanoseconds())
		dbg.Printf("1RTT: %s\n", latency)

		if callback != nil {
			callback(latency)
		}

		time.Sleep(echoFreq)
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	return latencies, nil
}

func prepareICMPPacket() []byte {
	icmpData := make([]byte, 8+echoOptionDataSize) // header + data
	icmpData[0] = 8                                // echo
	icmpData[1] = 0                                // code
	icmpData[2] = 0                                // checksum
	icmpData[3] = 0                                // checksum
	icmpData[4] = 0                                // id
	icmpData[5] = 1                                // id
	icmpData[6] = 0                                // seq
	icmpData[7] = 1                                // seq

	echoMessage := "Hi! SpeedTest-Go \\(●'◡'●)/"
	for i := range len(echoMessage) {
		icmpData[8+i] = echoMessage[i]
	}

	icmpData[8+echoOptionDataSize-1] = 6

	return icmpData
}

func (s *Server) sendOneICMPPing(dialContext interface {
	Write(data []byte) (n int, err error)
	Read(data []byte) (n int, err error)
	SetDeadline(t time.Time) error
	Close() error
}, icmpData []byte, _ int, readTimeout time.Duration,
) (time.Duration, error) {
	// Update checksum and seq
	icmpData[2] = 0
	icmpData[3] = 0
	icmpData[6] = byte(1 >> 8)
	icmpData[7] = byte(1)
	cs := checkSum(icmpData)
	icmpData[2] = byte(cs >> 8)
	icmpData[3] = byte(cs)

	sTime := time.Now()
	_ = dialContext.SetDeadline(sTime.Add(readTimeout))

	_, err := dialContext.Write(icmpData)
	if err != nil {
		return 0, fmt.Errorf("failed to write ICMP packet: %w", err)
	}

	buf := make([]byte, 20+echoOptionDataSize+8)

	_, err = dialContext.Read(buf)
	if err != nil || buf[20] != 0x00 {
		return 0, fmt.Errorf("failed to read ICMP response: %w", err)
	}

	return time.Since(sTime), nil
}

func checkSum(data []byte) uint16 {
	var sum uint16
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint16(data[i])<<8 + uint16(data[i+1])
	}

	if len(data)%2 == 1 {
		sum += uint16(data[len(data)-1]) << 8
	}

	return ^sum
}

// StandardDeviation calculates the mean, variance, standard deviation, min, and max of a vector.
func StandardDeviation(vector []int64) (int64, int64, int64, int64, int64) {
	if len(vector) == 0 {
		return 0, 0, 0, 0, 0
	}

	var (
		sumNum, accumulate                     int64
		mean, variance, stdDev, minVal, maxVal int64
	)

	minVal = math.MaxInt64
	maxVal = math.MinInt64

	for _, value := range vector {
		sumNum += value
		if minVal > value {
			minVal = value
		}

		if maxVal < value {
			maxVal = value
		}
	}

	mean = sumNum / int64(len(vector))
	for _, value := range vector {
		accumulate += (value - mean) * (value - mean)
	}

	variance = accumulate / int64(len(vector))
	stdDev = int64(math.Sqrt(float64(variance)))

	return mean, variance, stdDev, minVal, maxVal
}

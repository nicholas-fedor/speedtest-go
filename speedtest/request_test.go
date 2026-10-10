package speedtest

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

func TestServer_MultiDownloadTestContext(t *testing.T) {
	t.Parallel()

	type args struct {
		servers Servers
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{servers: Servers{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.MultiDownloadTestContext(ctx, tt.args.servers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_MultiUploadTestContext(t *testing.T) {
	t.Parallel()

	type args struct {
		servers Servers
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{servers: Servers{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.MultiUploadTestContext(ctx, tt.args.servers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_DownloadTest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.DownloadTest()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_DownloadTestContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.DownloadTestContext(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_downloadTestContext(t *testing.T) {
	t.Parallel()

	type args struct {
		downloadRequest downloadFunc
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.downloadTestContext(ctx, tt.args.downloadRequest)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_UploadTest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.UploadTest()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_UploadTestContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.UploadTestContext(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_uploadTestContext(t *testing.T) {
	t.Parallel()

	type args struct {
		uploadRequest uploadFunc
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.uploadTestContext(ctx, tt.args.uploadRequest)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_downloadRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		s *Server
		w int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			args:    args{s: nil, w: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := downloadRequest(ctx, tt.args.s, tt.args.w)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_uploadRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		s *Server
		w int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			args:    args{s: nil, w: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := uploadRequest(ctx, tt.args.s, tt.args.w)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_PingTest(t *testing.T) {
	t.Parallel()

	type args struct {
		callback func(latency time.Duration)
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.PingTest(tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_PingTestContext(t *testing.T) {
	t.Parallel()

	type args struct {
		callback func(latency time.Duration)
	}

	tests := []struct {
		name    string
		s       *Server
		args    args
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			err := tt.s.PingTestContext(ctx, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Server
		wantErr bool
	}{
		{
			name:    "nil server",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.s.TestAll()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_TCPPing(t *testing.T) {
	t.Parallel()

	type args struct {
		echoTimes int
		echoFreq  time.Duration
		callback  func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{echoTimes: 1, echoFreq: time.Second, callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.TCPPing(
				ctx,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func TestServer_HTTPPing(t *testing.T) {
	t.Parallel()

	type args struct {
		echoTimes int
		echoFreq  time.Duration
		callback  func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name:    "nil server",
			s:       nil,
			args:    args{echoTimes: 1, echoFreq: time.Second, callback: func(_ time.Duration) {}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.HTTPPing(
				ctx,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func TestServer_ICMPPing(t *testing.T) {
	t.Parallel()

	type args struct {
		readTimeout time.Duration
		echoTimes   int
		echoFreq    time.Duration
		callback    func(latency time.Duration)
	}

	tests := []struct {
		name          string
		s             *Server
		args          args
		wantLatencies []int64
		wantErr       bool
	}{
		{
			name: "nil server",
			s:    nil,
			args: args{
				readTimeout: time.Second,
				echoTimes:   1,
				echoFreq:    time.Second,
				callback:    func(_ time.Duration) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotLatencies, err := tt.s.ICMPPing(
				ctx,
				tt.args.readTimeout,
				tt.args.echoTimes,
				tt.args.echoFreq,
				tt.args.callback,
			)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLatencies, gotLatencies)
		})
	}
}

func Test_checkSum(t *testing.T) {
	t.Parallel()

	type args struct {
		data []byte
	}

	tests := []struct {
		name string
		args args
		want uint16
	}{
		{
			name: "checksum of empty data",
			args: args{data: []byte{}},
			want: 0xffff,
		},
		{
			name: "checksum of data",
			args: args{data: []byte{1, 2, 3, 4}},
			want: 0xfbf9,
		},
		{
			name: "RFC 1071 example folds the carry",
			args: args{data: []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}},
			want: 0x220d,
		},
		{
			name: "echo request sent by ICMPPing",
			args: args{data: prepareICMPPacket()},
			want: 0xc770,
		},
		{
			name: "odd length pads the last byte",
			args: args{data: []byte{0xab, 0xcd, 0xef}},
			want: 0x6531,
		},
		{
			name: "carry wraps more than once",
			args: args{data: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
			want: 0x0000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := checkSum(tt.args.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

// runningClient returns a client whose data manager accepts transferred bytes, as it does during a test.
func runningClient(t *testing.T) (*Speedtest, *DataManager) {
	t.Helper()

	client := New()

	manager, ok := client.Manager.(*DataManager)
	require.True(t, ok)

	manager.runningRW.Lock()
	manager.running = true
	manager.runningRW.Unlock()

	return client, manager
}

// Test_downloadRequest_FakeServer checks that a successful download returns no error and counts its bytes, and that
// an error status returns a status error without counting the error page or creating a chunk.
func Test_downloadRequest_FakeServer(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client, manager := runningClient(t)
		server := &Server{URL: api.ServerUploadURL("1001"), Context: client}

		require.NoError(t, downloadRequest(context.Background(), server, 0))

		assert.Equal(t, int64(testserver.DefaultDownloadSize), manager.GetTotalDownload())
	})

	t.Run("error status", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(
			testserver.PathDownload,
			testserver.Response{Status: http.StatusInternalServerError, Body: "error page"},
		)

		client, manager := runningClient(t)
		server := &Server{URL: api.ServerUploadURL("1001"), Context: client}

		err := downloadRequest(context.Background(), server, 0)
		require.ErrorIs(t, err, ErrUnexpectedStatus)

		assert.Zero(
			t,
			manager.GetTotalDownload(),
			"the error page must not count as downloaded data",
		)
		assert.Empty(t, *manager.Snapshot, "no chunk should be created for an error page")
	})
}

// TestServer_DownloadTestContext_ErrorStatus checks that a server answering downloads with an error status ends with
// a download speed of N/A instead of a rate measured from error pages.
func TestServer_DownloadTestContext_ErrorStatus(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(
		testserver.PathDownload,
		testserver.Response{Status: http.StatusInternalServerError, Body: "error page"},
	)

	client := New(WithUserConfig(&UserConfig{MaxConnections: 2}))
	client.SetCaptureTime(300 * time.Millisecond)

	server := &Server{URL: api.ServerUploadURL("1001"), Context: client}

	require.NoError(t, server.DownloadTestContext(context.Background()))

	assert.InDelta(
		t,
		-1,
		float64(server.DLSpeed),
		1e-9,
		"a test of only failed requests reports N/A",
	)
	assert.Zero(t, client.GetTotalDownload())
}

// TestServer_HTTPPing_ErrorStatus checks that error statuses fail the probe, and that the error names both the
// timeout and the status.
func TestServer_HTTPPing_ErrorStatus(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(
		testserver.PathLatency,
		testserver.Response{Status: http.StatusInternalServerError},
	)

	server := &Server{URL: api.ServerUploadURL("1001"), Context: New()}

	latencies, err := server.HTTPPing(context.Background(), 2, time.Millisecond, nil)

	require.ErrorIs(t, err, ErrConnectTimeout)
	require.ErrorIs(t, err, ErrUnexpectedStatus)
	assert.Nil(t, latencies)
}

// TestServer_Ping_NoHost checks that each ping reports a server URL without a host as errNoHost, instead of
// wrapping a nil parse error.
func TestServer_Ping_NoHost(t *testing.T) {
	t.Parallel()

	server := &Server{URL: "/speedtest/upload.php", Context: New()}

	tests := []struct {
		ping func() error
		name string
	}{
		{
			name: "TCP",
			ping: func() error {
				_, err := server.TCPPing(context.Background(), 1, time.Millisecond, nil)

				return err
			},
		},
		{
			name: "HTTP",
			ping: func() error {
				_, err := server.HTTPPing(context.Background(), 1, time.Millisecond, nil)

				return err
			},
		},
		{
			name: "ICMP",
			ping: func() error {
				_, err := server.ICMPPing(
					context.Background(),
					time.Second,
					1,
					time.Millisecond,
					nil,
				)

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.ping()

			require.ErrorIs(t, err, errNoHost)
			assert.NotContains(t, err.Error(), "%!w", "the error must not wrap a nil cause")
			assert.Contains(t, err.Error(), server.URL)
		})
	}
}

// TestServer_HTTPPing_Success checks that each requested echo after the warm-up yields a latency.
func TestServer_HTTPPing_Success(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &Server{URL: api.ServerUploadURL("1001"), Context: New()}

	latencies, err := server.HTTPPing(context.Background(), 3, time.Millisecond, nil)
	require.NoError(t, err)

	assert.Len(t, latencies, 3)
}

// Test_sleepContext checks that the wait runs its full length, or ends promptly with the context's error.
func Test_sleepContext(t *testing.T) {
	t.Parallel()

	require.NoError(t, sleepContext(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)

	start := time.Now()
	err := sleepContext(ctx, time.Minute)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), time.Second)

	// With a context that already ended and a timer that is ready at once, both select cases are ready.
	for range 100 {
		require.ErrorIs(t, sleepContext(ctx, 0), context.Canceled)
	}
}

// TestServer_Ping_CancelBetweenEchoes checks that cancelling a ping between echoes ends it at once with the
// context's error and the latencies measured so far, instead of finishing the wait and the remaining echoes.
func TestServer_Ping_CancelBetweenEchoes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ping func(*Server, context.Context, func(time.Duration)) ([]int64, error)
		name string
	}{
		{
			name: "TCP",
			ping: func(s *Server, ctx context.Context, callback func(time.Duration)) ([]int64, error) {
				return s.TCPPing(ctx, 10, 200*time.Millisecond, callback)
			},
		},
		{
			name: "HTTP",
			ping: func(s *Server, ctx context.Context, callback func(time.Duration)) ([]int64, error) {
				return s.HTTPPing(ctx, 10, 200*time.Millisecond, callback)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tcp := testserver.NewTCPServer(t)
			api := testserver.NewAPI(t)
			server := &Server{URL: api.ServerUploadURL("1001"), Host: tcp.Addr(), Context: New()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Measure from the cancel, since the HTTP ping waits once after its unmeasured warm-up request.
			var cancelledAt time.Time

			latencies, err := tt.ping(server, ctx, func(time.Duration) {
				cancelledAt = time.Now()

				cancel()
			})

			require.ErrorIs(t, err, context.Canceled)
			assert.Len(t, latencies, 1, "the echo measured before cancelling is kept")
			assert.Less(
				t,
				time.Since(cancelledAt),
				100*time.Millisecond,
				"the ping must not finish its wait",
			)
		})
	}
}

// TestServer_PingTestContext_Cancelled checks that a cancelled ping keeps the server's earlier latency results.
func TestServer_PingTestContext_Cancelled(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &Server{
		URL:     api.ServerUploadURL("1001"),
		Latency: 42 * time.Millisecond,
		Context: New(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, server.PingTestContext(ctx, nil), context.Canceled)

	assert.Equal(t, 42*time.Millisecond, server.Latency)
	assert.Nil(t, server.TestDuration.Ping)
}

// TestServer_ICMPPing_CancelDuringRead checks that cancelling a ping whose echo is never answered ends the pending
// read at once, instead of after readTimeout. It needs raw ICMP sockets and an unanswered echo, so it skips unless
// run privileged with echo replies disabled, such as in a network namespace with net.ipv4.icmp_echo_ignore_all=1.
func TestServer_ICMPPing_CancelDuringRead(t *testing.T) {
	t.Parallel()

	server := &Server{URL: "http://127.0.0.1/speedtest/upload.php", Context: New()}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	_, err := server.ICMPPing(ctx, 5*time.Second, 1, time.Millisecond, nil)

	switch {
	case err != nil && strings.Contains(err.Error(), "failed to dial ICMP"):
		t.Skipf("raw ICMP sockets need privileges: %v", err)
	case err == nil:
		t.Skip("loopback answered the echo, so no read was pending when the context ended")
	}

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), time.Second, "the read must not wait for its 5s timeout")
}

// TestServer_SpeedTests_Cancel checks that cancelling a download or upload ends it well before the capture time
// with the context's error, and leaves the server's earlier results untouched.
func TestServer_SpeedTests_Cancel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		run  func(*Server, context.Context) error
		name string
	}{
		{name: "download", run: (*Server).DownloadTestContext},
		{name: "upload", run: (*Server).UploadTestContext},
		{
			name: "multi download",
			run: func(s *Server, ctx context.Context) error {
				return s.MultiDownloadTestContext(ctx, Servers{s})
			},
		},
		{
			name: "multi upload",
			run: func(s *Server, ctx context.Context) error {
				return s.MultiUploadTestContext(ctx, Servers{s})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			client := New(WithUserConfig(&UserConfig{MaxConnections: 2}))
			server := &Server{
				ID:      "1001",
				URL:     api.ServerUploadURL("1001"),
				Latency: time.Millisecond,
				DLSpeed: 42,
				ULSpeed: 42,
				Context: client,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			start := time.Now()
			err := tt.run(server, ctx)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Less(t, time.Since(start), 2*time.Second, "the default capture time is 15s")
			assert.InDelta(
				t,
				42,
				float64(server.DLSpeed),
				0,
				"a cancelled test must not change the result",
			)
			assert.InDelta(
				t,
				42,
				float64(server.ULSpeed),
				0,
				"a cancelled test must not change the result",
			)
			assert.Nil(t, server.TestDuration.Download)
			assert.Nil(t, server.TestDuration.Upload)
		})
	}
}

// Test_checkSum_Verifies checks the property receivers rely on: a message carrying its checksum sums to zero.
func Test_checkSum_Verifies(t *testing.T) {
	t.Parallel()

	packet := prepareICMPPacket()

	sum := checkSum(packet)
	packet[2], packet[3] = byte(sum>>8), byte(sum)

	assert.Zero(t, checkSum(packet))
}

// TestServer_TCPPing_FakeServer checks that a client from New can TCP-ping a server. The dialer used to be nil
// unless a source address was configured, which made this call panic.
func TestServer_TCPPing_FakeServer(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)
	server := &Server{Host: tcp.Addr(), Context: New()}

	var callbacks int

	latencies, err := server.TCPPing(
		context.Background(),
		3,
		time.Millisecond,
		func(time.Duration) {
			callbacks++
		},
	)
	require.NoError(t, err)

	assert.Len(t, latencies, 3)
	assert.Equal(t, 3, callbacks)

	var pings int

	for _, command := range tcp.Commands() {
		if strings.HasPrefix(command.Line, "PING ") {
			pings++
		}
	}

	assert.Equal(t, 6, pings, "each echo sends two PING commands")
}

// TestServer_TCPPing_ClosesConnection checks that TCP ping ends its session with QUIT and closes the socket, so
// pinging every listed server does not leave one connection open per server.
func TestServer_TCPPing_ClosesConnection(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)
	server := &Server{Host: tcp.Addr(), Context: New()}

	_, err := server.TCPPing(context.Background(), 1, time.Millisecond, nil)
	require.NoError(t, err)

	// The server closes its side only after reading everything sent before the client closed, including QUIT.
	require.Eventually(
		t,
		func() bool { return tcp.OpenConns() == 0 },
		5*time.Second,
		10*time.Millisecond,
	)

	commands := tcp.Commands()
	require.NotEmpty(t, commands)
	assert.Equal(t, "QUIT", commands[len(commands)-1].Line)
}

// TestServer_TCPPing_Refused checks that pinging a closed port reports an error instead of panicking.
func TestServer_TCPPing_Refused(t *testing.T) {
	t.Parallel()

	var config net.ListenConfig

	listener, err := config.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	server := &Server{Host: addr, Context: New()}

	_, err = server.TCPPing(context.Background(), 1, time.Millisecond, nil)
	require.Error(t, err)
}

// TestServer_ICMPPing_Loopback checks that ICMP ping no longer panics on a nil dialer, and that a privileged run
// gets a reply from loopback. The kernel only answers a request with a valid checksum, and the client must skip its
// own request, which a raw socket also receives on loopback. Unprivileged runs fail to open the raw socket.
func TestServer_ICMPPing_Loopback(t *testing.T) {
	t.Parallel()

	server := &Server{URL: "http://127.0.0.1/speedtest/upload.php", Context: New()}

	var (
		latencies []int64
		err       error
	)

	require.NotPanics(t, func() {
		latencies, err = server.ICMPPing(
			context.Background(),
			time.Second,
			1,
			time.Millisecond,
			nil,
		)
	})

	switch {
	case err == nil:
		assert.Len(t, latencies, 1)
	case strings.Contains(err.Error(), "failed to dial ICMP"):
		t.Logf("raw ICMP sockets need privileges: %v", err)
	case runtime.GOOS == "linux":
		require.NoError(t, err, "a privileged Linux run should receive the loopback echo reply")
	default:
		t.Skipf("raw ICMP on %s: %v", runtime.GOOS, err)
	}
}

// Test_isEchoReply checks which packets read from a raw ICMP socket count as the reply to a request.
func Test_isEchoReply(t *testing.T) {
	t.Parallel()

	request := prepareICMPPacket()

	// ipv4 prefixes an ICMP message with a minimal IPv4 header, or one with options when optionWords is set.
	ipv4 := func(optionWords int, icmp []byte) []byte {
		headerLen := 20 + 4*optionWords

		packet := make([]byte, headerLen+len(icmp))
		packet[0] = 0x40 | byte(5+optionWords)
		copy(packet[headerLen:], icmp)

		return packet
	}

	reply := slices.Clone(request)
	reply[0] = icmpEchoReply

	otherPing := slices.Clone(reply)
	otherPing[5] = 0x99

	tests := []struct {
		name   string
		packet []byte
		want   bool
	}{
		{name: "matching reply", packet: ipv4(0, reply), want: true},
		{name: "reply after IP options", packet: ipv4(2, reply), want: true},
		{name: "own request on loopback", packet: ipv4(0, request)},
		{name: "reply to another ping", packet: ipv4(0, otherPing)},
		{name: "truncated message", packet: ipv4(0, reply[:4])},
		{name: "header longer than packet", packet: ipv4(0, reply)[:10]},
		{name: "empty packet", packet: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isEchoReply(tt.packet, request))
		})
	}
}

func TestStandardDeviation(t *testing.T) {
	t.Parallel()

	type args struct {
		vector []int64
	}

	tests := []struct {
		name         string
		args         args
		wantMean     int64
		wantVariance int64
		wantStdDev   int64
		wantMin      int64
		wantMax      int64
	}{
		{
			name:         "standard deviation of single value",
			args:         args{vector: []int64{5}},
			wantMean:     5,
			wantVariance: 0,
			wantStdDev:   0,
			wantMin:      5,
			wantMax:      5,
		},
		{
			name:         "standard deviation of multiple values",
			args:         args{vector: []int64{1, 2, 3, 4, 5}},
			wantMean:     3,
			wantVariance: 2,
			wantStdDev:   1,
			wantMin:      1,
			wantMax:      5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMean, gotVariance, gotStdDev, gotMin, gotMax := StandardDeviation(tt.args.vector)
			assert.Equal(t, tt.wantMean, gotMean)
			assert.Equal(t, tt.wantVariance, gotVariance)
			assert.Equal(t, tt.wantStdDev, gotStdDev)
			assert.Equal(t, tt.wantMin, gotMin)
			assert.Equal(t, tt.wantMax, gotMax)
		})
	}
}

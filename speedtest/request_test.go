package speedtest

import (
	"context"
	"net"
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

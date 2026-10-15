// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"math"
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

// Test_checkSum checks the RFC 1071 checksum of ICMP messages, including an odd length and carries.
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
			name: "echo request sent by icmpPing",
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

// TestClient_httpPing_ErrorStatus checks that error statuses fail the probe, and that the error names both the
// timeout and the status.
func TestClient_httpPing_ErrorStatus(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(
		testserver.PathLatency,
		testserver.Response{Status: http.StatusInternalServerError},
	)

	server := &Server{URL: api.ServerUploadURL("1001")}

	latencies, err := newTestClient(
		t,
	).httpPing(context.Background(), server, 2, time.Millisecond, nil)

	require.ErrorIs(t, err, ErrConnectTimeout)
	require.ErrorIs(t, err, ErrUnexpectedStatus)
	assert.Nil(t, latencies)
}

// TestClient_Ping_NoHost checks that each ping reports a server URL without a host as errNoHost, instead of
// wrapping a nil parse error.
func TestClient_Ping_NoHost(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	server := &Server{URL: "/speedtest/upload.php"}

	tests := []struct {
		ping func() error
		name string
	}{
		{
			name: "TCP",
			ping: func() error {
				_, err := client.tcpPing(context.Background(), server, 1, time.Millisecond, nil)

				return err
			},
		},
		{
			name: "HTTP",
			ping: func() error {
				_, err := client.httpPing(context.Background(), server, 1, time.Millisecond, nil)

				return err
			},
		},
		{
			name: "ICMP",
			ping: func() error {
				_, err := client.icmpPing(
					context.Background(),
					server,
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

// TestClient_httpPing_Success checks that each requested echo after the warm-up yields a latency.
func TestClient_httpPing_Success(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &Server{URL: api.ServerUploadURL("1001")}

	latencies, err := newTestClient(
		t,
	).httpPing(context.Background(), server, 3, time.Millisecond, nil)
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

// TestClient_Ping_CancelBetweenEchoes checks that cancelling a ping between echoes ends it at once with the
// context's error and the latencies measured so far, instead of finishing the wait and the remaining echoes.
func TestClient_Ping_CancelBetweenEchoes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ping func(*Client, context.Context, *Server, func(time.Duration)) ([]int64, error)
		name string
	}{
		{
			name: "TCP",
			ping: func(c *Client, ctx context.Context, s *Server, callback func(time.Duration)) ([]int64, error) {
				return c.tcpPing(ctx, s, 10, 200*time.Millisecond, callback)
			},
		},
		{
			name: "HTTP",
			ping: func(c *Client, ctx context.Context, s *Server, callback func(time.Duration)) ([]int64, error) {
				return c.httpPing(ctx, s, 10, 200*time.Millisecond, callback)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tcp := testserver.NewTCPServer(t)
			api := testserver.NewAPI(t)
			server := &Server{URL: api.ServerUploadURL("1001"), Host: tcp.Addr()}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Measure from the cancel, since the HTTP ping waits once after its unmeasured warm-up request.
			var cancelledAt time.Time

			latencies, err := tt.ping(newTestClient(t), ctx, server, func(time.Duration) {
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

// TestClient_Ping_Cancelled checks that a cancelled ping fails with the context's error and leaves the server as it
// was.
func TestClient_Ping_Cancelled(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &Server{URL: api.ServerUploadURL("1001"), Latency: 42 * time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stats, err := newTestClient(t).Ping(ctx, server)
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, PingStats{}, stats)
	assert.Equal(t, 42*time.Millisecond, server.Latency)
}

// TestClient_icmpPing_CancelDuringRead checks that cancelling a ping whose echo is never answered ends the pending
// read at once, instead of after readTimeout. It needs raw ICMP sockets and an unanswered echo, so it skips unless
// run privileged with echo replies disabled, such as in a network namespace with net.ipv4.icmp_echo_ignore_all=1.
func TestClient_icmpPing_CancelDuringRead(t *testing.T) {
	t.Parallel()

	server := &Server{URL: "http://127.0.0.1/speedtest/upload.php"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	_, err := newTestClient(t).icmpPing(ctx, server, 5*time.Second, 1, time.Millisecond, nil)

	switch {
	case err != nil && strings.Contains(err.Error(), "failed to dial ICMP"):
		t.Skipf("raw ICMP sockets need privileges: %v", err)
	case err == nil:
		t.Skip("loopback answered the echo, so no read was pending when the context ended")
	}

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), time.Second, "the read must not wait for its 5s timeout")
}

// Test_checkSum_Verifies checks the property receivers rely on: a message carrying its checksum sums to zero.
func Test_checkSum_Verifies(t *testing.T) {
	t.Parallel()

	packet := prepareICMPPacket()

	sum := checkSum(packet)
	packet[2], packet[3] = byte(sum>>8), byte(sum)

	assert.Zero(t, checkSum(packet))
}

// TestClient_tcpPing_FakeServer checks that a client from New can TCP-ping a server, which needs its dialer.
func TestClient_tcpPing_FakeServer(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)
	server := &Server{Host: tcp.Addr()}

	var callbacks int

	latencies, err := newTestClient(t).tcpPing(
		context.Background(),
		server,
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

// TestClient_tcpPing_ClosesConnection checks that TCP ping ends its session with QUIT and closes the socket, so
// pinging every listed server does not leave one connection open per server.
func TestClient_tcpPing_ClosesConnection(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)
	server := &Server{Host: tcp.Addr()}

	_, err := newTestClient(t).tcpPing(context.Background(), server, 1, time.Millisecond, nil)
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

// TestClient_tcpPing_Refused checks that pinging a closed port reports an error instead of panicking.
func TestClient_tcpPing_Refused(t *testing.T) {
	t.Parallel()

	var config net.ListenConfig

	listener, err := config.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	server := &Server{Host: addr}

	_, err = newTestClient(t).tcpPing(context.Background(), server, 1, time.Millisecond, nil)
	require.Error(t, err)
}

// TestClient_icmpPing_Loopback checks that ICMP ping does not panic, and that a privileged run gets a reply from
// loopback. The kernel only answers a request with a valid checksum, and the client must skip its
// own request, which a raw socket also receives on loopback. Unprivileged runs fail to open the raw socket.
func TestClient_icmpPing_Loopback(t *testing.T) {
	t.Parallel()

	client := newTestClient(t)
	server := &Server{URL: "http://127.0.0.1/speedtest/upload.php"}

	var (
		latencies []int64
		err       error
	)

	require.NotPanics(t, func() {
		latencies, err = client.icmpPing(
			context.Background(),
			server,
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

// TestStandardDeviation checks the summary statistics of a vector, including an empty one.
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

// TestClient_Ping checks that a ping measures the requested number of round trips over HTTP, reports each to the
// callback, and summarizes them, without changing the server.
func TestClient_Ping(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Latency: 42 * time.Millisecond}
	before := *server

	var reported []time.Duration

	stats, err := newTestClient(t).Ping(
		context.Background(),
		server,
		WithPingCount(3),
		WithPingInterval(0),
		WithLatencyCallback(func(latency time.Duration) { reported = append(reported, latency) }),
	)
	require.NoError(t, err)

	require.Len(t, stats.Latencies, 3)
	assert.Equal(t, stats.Latencies, reported)
	assert.LessOrEqual(t, stats.Min, stats.Latency)
	assert.LessOrEqual(t, stats.Latency, stats.Max)
	assert.Positive(t, stats.Duration)
	assert.Equal(t, before, *server, "a ping must not change the server")
}

// TestClient_Ping_TCPMode checks that the TCP ping mode measures round trips with the TCP protocol.
func TestClient_Ping_TCPMode(t *testing.T) {
	t.Parallel()

	tcp := testserver.NewTCPServer(t)

	stats, err := newTestClient(t, WithPingMode(TCP)).Ping(
		context.Background(),
		&Server{Host: tcp.Addr()},
		WithPingCount(2),
		WithPingInterval(0),
	)
	require.NoError(t, err)

	assert.Len(t, stats.Latencies, 2)
	assert.Positive(t, countPrefix(tcp, "PING "))
}

// TestClient_Ping_InvalidOptions checks that a count below 1 or a negative interval is rejected before anything is
// sent.
func TestClient_Ping_InvalidOptions(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t)
	server := &Server{URL: api.ServerUploadURL("1001")}

	_, err := client.Ping(context.Background(), server, WithPingCount(0))
	require.ErrorIs(t, err, ErrInvalidOption)

	_, err = client.Ping(context.Background(), server, WithPingInterval(-time.Second))
	require.ErrorIs(t, err, ErrInvalidOption)

	assert.False(t, api.Requested(testserver.PathLatency))
}

// TestClient_Ping_Unreachable checks that a ping whose every round trip fails reports ErrConnectTimeout.
func TestClient_Ping_Unreachable(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(
		testserver.PathLatency,
		testserver.Response{Status: http.StatusServiceUnavailable},
	)

	_, err := newTestClient(t).Ping(
		context.Background(),
		&Server{URL: api.ServerUploadURL("1001")},
		WithPingCount(2),
		WithPingInterval(0),
	)
	require.ErrorIs(t, err, ErrConnectTimeout)
}

// TestStandardDeviation_LargeSpread checks that latencies seconds apart keep a correct standard deviation instead of
// overflowing, and that a variance beyond int64 is clamped.
func TestStandardDeviation_LargeSpread(t *testing.T) {
	t.Parallel()

	mean, variance, stdDev, minValue, maxValue := StandardDeviation(
		[]int64{0, int64(10 * time.Second)},
	)

	assert.Equal(t, int64(5*time.Second), mean)
	assert.Equal(
		t,
		int64(math.MaxInt64),
		variance,
		"a variance of 2.5e19 ns² does not fit in int64",
	)
	assert.Equal(t, int64(5*time.Second), stdDev)
	assert.Zero(t, minValue)
	assert.Equal(t, int64(10*time.Second), maxValue)
}

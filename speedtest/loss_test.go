package speedtest

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
)

func TestNewPacketLossAnalyzer(t *testing.T) {
	t.Parallel()

	type args struct {
		options *PacketLossAnalyzerOptions
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "with nil options",
			args: args{options: nil},
		},
		{
			name: "with custom options",
			args: args{options: &PacketLossAnalyzerOptions{
				SamplingDuration: time.Second * 10,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewPacketLossAnalyzer(tt.args.options)
			assert.NotNil(t, got)
			assert.NotNil(t, got.options)

			if tt.args.options != nil && tt.args.options.SamplingDuration != 0 {
				assert.Equal(t, time.Second*10, got.options.SamplingDuration)
			}
		})
	}
}

func TestPacketLossAnalyzer_RunMulti(t *testing.T) {
	t.Parallel()

	type args struct {
		hosts []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty hosts",
			args:    args{hosts: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid host",
			args:    args{hosts: []string{"invalid"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)

			got, err := pla.RunMulti(tt.args.hosts)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestPacketLossAnalyzer_RunMultiWithContext(t *testing.T) {
	t.Parallel()

	type args struct {
		hosts []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty hosts with context",
			args:    args{hosts: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid host with context",
			args:    args{hosts: []string{"invalid"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx := context.Background()

			got, err := pla.RunMultiWithContext(ctx, tt.args.hosts)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestPacketLossAnalyzer_Run(t *testing.T) {
	t.Parallel()

	type args struct {
		host     string
		callback func(packetLoss *transport.PLoss)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid host",
			args: args{
				host:     "invalid",
				callback: func(_ *transport.PLoss) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)

			err := pla.Run(tt.args.host, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPacketLossAnalyzer_RunWithContext(t *testing.T) {
	t.Parallel()

	type args struct {
		host     string
		callback func(packetLoss *transport.PLoss)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid host with context",
			args: args{
				host:     "invalid",
				callback: func(_ *transport.PLoss) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx := context.Background()

			err := pla.RunWithContext(ctx, tt.args.host, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPacketLossAnalyzer_loopSampler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "loop sampler with cancelled context",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			require.NoError(t, pla.loopSampler(ctx, nil, func(_ *transport.PLoss) {}))
		})
	}
}

func TestPacketLossAnalyzer_loopSender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "loop sender with cancelled context",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()                 // Cancel immediately
			pla.loopSender(ctx, nil) // Should return immediately since context is cancelled
		})
	}
}

// TestNewPacketLossAnalyzer_Source checks that the default dialers bind to an IP source and stay unbound otherwise,
// and that dialers given in the options are kept.
func TestNewPacketLossAnalyzer_Source(t *testing.T) {
	t.Parallel()

	loopback := net.IPv4(127, 0, 0, 1)

	tests := []struct {
		wantIP net.IP
		name   string
		source string
	}{
		{name: "IP", source: "127.0.0.1", wantIP: loopback},
		{name: "IP with network prefix", source: "udp://127.0.0.1", wantIP: loopback},
		{name: "interface name", source: "eth0"},
		{name: "empty", source: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options := NewPacketLossAnalyzer(
				&PacketLossAnalyzerOptions{SourceInterface: tt.source},
			).options

			if tt.wantIP == nil {
				assert.Nil(t, options.TCPDialer.LocalAddr)
				assert.Nil(t, options.UDPDialer.LocalAddr)

				return
			}

			assert.Equal(t, &net.TCPAddr{IP: tt.wantIP}, options.TCPDialer.LocalAddr)
			assert.Equal(t, &net.UDPAddr{IP: tt.wantIP}, options.UDPDialer.LocalAddr)
		})
	}

	t.Run("given dialers are kept", func(t *testing.T) {
		t.Parallel()

		tcpDialer, udpDialer := &net.Dialer{}, &net.Dialer{}
		options := NewPacketLossAnalyzer(&PacketLossAnalyzerOptions{
			SourceInterface: "127.0.0.1",
			TCPDialer:       tcpDialer,
			UDPDialer:       udpDialer,
		}).options

		assert.Same(t, tcpDialer, options.TCPDialer)
		assert.Same(t, udpDialer, options.UDPDialer)
		assert.Nil(t, tcpDialer.LocalAddr)
	})
}

// fastAnalyzer returns an analyzer that samples and sends quickly, bound to the loopback source.
func fastAnalyzer() *PacketLossAnalyzer {
	return NewPacketLossAnalyzer(&PacketLossAnalyzerOptions{
		RemoteSamplingInterval: 20 * time.Millisecond,
		PacketSendingInterval:  2 * time.Millisecond,
		SourceInterface:        "127.0.0.1",
	})
}

// sampleRecorder collects packet-loss samples from the analyzer's callback.
type sampleRecorder struct {
	samples []transport.PLoss
	mu      sync.Mutex
}

// record stores one sample.
func (r *sampleRecorder) record(packetLoss *transport.PLoss) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.samples = append(r.samples, *packetLoss)
}

// all returns a copy of the samples so far.
func (r *sampleRecorder) all() []transport.PLoss {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]transport.PLoss(nil), r.samples...)
}

// registeredUUID returns the UUID the analyzer registered with the fake server.
func registeredUUID(t *testing.T, srv *testserver.TCPServer) string {
	t.Helper()

	for _, command := range srv.Commands() {
		if uuid, found := strings.CutPrefix(command.Line, "HI "); found {
			return uuid
		}
	}

	require.Fail(t, "the analyzer never registered a UUID")

	return ""
}

// countCommands returns how many commands the fake server received with the given line.
func countCommands(srv *testserver.TCPServer, line string) int {
	var count int

	for _, command := range srv.Commands() {
		if command.Line == line {
			count++
		}
	}

	return count
}

// Test_endedByContext checks which sampling errors mark the normal end of a run, including a deadline error that
// arrives before the context reports its own deadline.
func Test_endedByContext(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	deadlineErr := fmt.Errorf("operation ended by context: %w", context.DeadlineExceeded)

	assert.True(t, endedByContext(cancelled, io.EOF), "a done context ends the run")
	assert.True(
		t,
		endedByContext(context.Background(), deadlineErr),
		"a deadline error ends the run",
	)
	assert.False(
		t,
		endedByContext(context.Background(), io.EOF),
		"a broken connection is a failure",
	)
}

// TestPacketLossAnalyzer_RunWithContext_FakeServer checks a full run: every PLOSS request yields a sample, the run
// ends cleanly with its context, and it leaves no connection or send loop behind.
func TestPacketLossAnalyzer_RunWithContext_FakeServer(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	recorder := &sampleRecorder{}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	require.NoError(t, fastAnalyzer().RunWithContext(ctx, srv.Addr(), recorder.record))

	samples := recorder.all()
	require.NotEmpty(t, samples)

	// The context may end while the last request is in flight, so at most that one may lack a sample.
	assert.GreaterOrEqual(
		t,
		len(samples),
		countCommands(srv, "PLOSS")-1,
		"every completed PLOSS request must yield a sample",
	)
	assert.Positive(t, samples[len(samples)-1].Sent)

	for i := 1; i < len(samples); i++ {
		assert.GreaterOrEqual(
			t,
			samples[i].Sent,
			samples[i-1].Sent,
			"counts only grow during a run",
		)
	}

	require.Eventually(
		t,
		func() bool { return srv.OpenConns() == 0 },
		5*time.Second,
		10*time.Millisecond,
	)

	// Count everything the run sent before it returned, then check that nothing more arrives afterwards.
	uuid := registeredUUID(t, srv)

	srv.SyncPackets(t)
	sent := srv.LossPackets(uuid)

	time.Sleep(50 * time.Millisecond)
	srv.SyncPackets(t)

	assert.Equal(t, sent, srv.LossPackets(uuid), "the send loop must stop when the run ends")
}

// TestPacketLossAnalyzer_RunWithContext_Loss checks that datagrams the server never receives show up as loss.
func TestPacketLossAnalyzer_RunWithContext_Loss(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	srv.SetFaults(testserver.Faults{LossDropEvery: 2})

	recorder := &sampleRecorder{}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	require.NoError(t, fastAnalyzer().RunWithContext(ctx, srv.Addr(), recorder.record))

	samples := recorder.all()
	require.NotEmpty(t, samples)

	assert.InDelta(t, 0.5, samples[len(samples)-1].Loss(), 0.2, "every second datagram is dropped")
}

// TestPacketLossAnalyzer_RunWithContext_ConnectionDropped checks that a sampler connection lost during the run ends
// it with an error instead of polling a dead connection until the context ends.
func TestPacketLossAnalyzer_RunWithContext_ConnectionDropped(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	srv.SetFaults(testserver.Faults{DropAfter: 4})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := fastAnalyzer().RunWithContext(ctx, srv.Addr(), func(*transport.PLoss) {})

	require.ErrorIs(t, err, io.EOF)
	require.NotErrorIs(
		t,
		err,
		transport.ErrUnsupported,
		"a failure after setup is not a setup failure",
	)
	assert.Less(t, time.Since(start), time.Second)
}

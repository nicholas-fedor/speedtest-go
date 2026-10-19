// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// Test_pingFormat checks the PING command for zero, positive, and negative timestamps.
func Test_pingFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		locTime int64
		want    []byte
	}{
		{
			name:    "zero time",
			locTime: 0,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '0'},
		},
		{
			name:    "positive time",
			locTime: 123,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '1', '2', '3'},
		},
		{
			name:    "negative time",
			locTime: -456,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '-', '4', '5', '6'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pingFormat(tt.locTime)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNewClient checks that a new client keeps its dialer, including a nil one, and gets an ID.
func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dialer *net.Dialer
	}{
		{
			name:   "nil dialer",
			dialer: nil,
		},
		{
			name:   "with dialer",
			dialer: &net.Dialer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(tt.dialer)
			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotEmpty(t, got.ID())
			assert.Equal(t, tt.dialer, got.dialer)
		})
	}
}

// TestClient_ID checks that ID returns the client's UUID.
func TestClient_ID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "empty id",
			id:   "",
			want: "",
		},
		{
			name: "test id",
			id:   "test-uuid",
			want: "test-uuid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				id: tt.id,
			}
			assert.Equal(t, tt.want, client.ID())
		})
	}
}

// TestClient_Connect checks that Connect records the host and opens a buffered connection, and that an invalid address
// is an error.
func TestClient_Connect(t *testing.T) {
	t.Parallel()

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	dialer := &net.Dialer{}
	client, err := NewClient(dialer)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("successful connect", func(t *testing.T) {
		t.Parallel()

		err := client.Connect(ctx, listener.Addr().String())
		require.NoError(t, err)
		assert.NotNil(t, client.conn)
		assert.Equal(t, listener.Addr().String(), client.host)
		assert.NotNil(t, client.reader)
	})

	t.Run("connect to invalid host", func(t *testing.T) {
		t.Parallel()

		client2, err := NewClient(dialer)
		require.NoError(t, err)
		err = client2.Connect(ctx, "invalid:99999")
		require.Error(t, err)
	})
}

// TestClient_Disconnect checks that Disconnect sends QUIT and drops the connection and its reader.
func TestClient_Disconnect(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	done := make(chan bool, 1)

	go func() {
		buf := make([]byte, 10)
		_, _ = client2.Read(buf) // consume the quit message

		done <- true
	}()

	err := client.Disconnect()
	require.NoError(t, err)
	assert.Nil(t, client.conn)
	assert.Nil(t, client.reader)

	<-done // wait for read
}

// TestClient_Write checks that Write adds the newline, and that writing without a connection returns ErrEmptyConn.
func TestClient_Write(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	t.Cleanup(func() { _ = client2.Close() })

	client := &Client{
		conn: client1,
	}

	t.Run("successful write", func(t *testing.T) {
		t.Parallel()

		done := make(chan string, 1)

		go func() {
			buf := make([]byte, 20)

			n, _ := client2.Read(buf)
			done <- string(buf[:n])
		}()

		err := client.Write([]byte("test data"))
		require.NoError(t, err)

		received := <-done
		assert.Equal(t, "test data\n", received)
	})

	t.Run("write with nil conn", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		err := client.Write([]byte("test"))
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

// TestClient_Read checks that Read returns one line with its newline, and that reading without a connection returns
// ErrEmptyConn.
func TestClient_Read(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	t.Cleanup(func() { _ = client2.Close() })

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	t.Run("successful read", func(t *testing.T) {
		t.Parallel()

		done := make(chan error, 1)

		go func() {
			_, err := client2.Write([]byte("response\n"))
			done <- err
		}()

		got, err := client.Read()
		require.NoError(t, err)
		assert.Equal(t, []byte("response\n"), got)

		assert.NoError(t, <-done)
	})

	t.Run("read with nil conn", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		_, err := client.Read()
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

// TestClient_Version checks the cached and fetched versions, and the "unknown" version for a reply that is not HELLO or
// carries no version.
func TestClient_Version(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		writeHI  bool
		response string
		want     string
	}{
		{
			name:     "cached version",
			version:  "1.2.3",
			writeHI:  false,
			response: "",
			want:     "1.2.3",
		},
		{
			name:     "fetch version",
			version:  "",
			writeHI:  true,
			response: "HELLO 2.11 (2.11.5) 2026-06-04.1737.7d0993d\n",
			want:     "2.11 (2.11.5) 2026-06-04.1737.7d0993d",
		},
		{
			name:     "reply that is not HELLO",
			version:  "",
			writeHI:  true,
			response: "HI 1.2.3\n",
			want:     "unknown",
		},
		{
			name:     "HELLO without a version",
			version:  "",
			writeHI:  true,
			response: "HELLO \n",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client1, client2 := net.Pipe()

			defer func() { _ = client2.Close() }()

			client := &Client{
				conn:    client1,
				reader:  bufio.NewReader(client1),
				version: tt.version,
			}

			if tt.writeHI {
				go func() {
					buf := make([]byte, 10)
					_, _ = client2.Read(buf) // consume HI\n
					_, _ = client2.Write([]byte(tt.response))
				}()
			}

			got := client.Version()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestClient_PingContext checks that pinging without a connection returns ErrEmptyConn.
func TestClient_PingContext(t *testing.T) {
	t.Parallel()

	t.Run("no connection", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		_, err := client.PingContext(context.Background())
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

// TestClient_InitPacketLoss checks that the packet-loss setup without a connection is an error.
func TestClient_InitPacketLoss(t *testing.T) {
	t.Parallel()

	client := &Client{
		id: "test-id",
	}

	// Test with no conn, should return error
	err := client.InitPacketLoss()
	assert.Error(t, err)
}

// TestClient_PacketLoss checks that a PLOSS reply becomes the packet-loss counts.
func TestClient_PacketLoss(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	// Simulate server response
	go func() {
		buf := make([]byte, 10)
		_, _ = client2.Read(buf) // read PLOSS\n
		_, _ = client2.Write([]byte("PLOSS 90 5 100\n"))
	}()

	got, err := client.PacketLoss()
	require.NoError(t, err)
	assert.Equal(t, &PLoss{Sent: 90, Dup: 5, Max: 100}, got)
}

// connectFake connects a new client to the fake TCP server and disconnects it when the test ends.
func connectFake(t *testing.T, srv *testserver.TCPServer) *Client {
	t.Helper()

	client, err := NewClient(&net.Dialer{Timeout: 5 * time.Second})
	require.NoError(t, err)
	require.NoError(t, client.Connect(context.Background(), srv.Addr()))

	t.Cleanup(func() { _ = client.Disconnect() })

	return client
}

// lastCommand returns the last command line the fake TCP server received, or an empty string.
func lastCommand(srv *testserver.TCPServer) string {
	commands := srv.Commands()
	if len(commands) == 0 {
		return ""
	}

	return commands[len(commands)-1].Line
}

// TestClient_Connect_NilDialer checks that a client without a dialer reports it instead of panicking.
func TestClient_Connect_NilDialer(t *testing.T) {
	t.Parallel()

	client, err := NewClient(nil)
	require.NoError(t, err)

	require.ErrorIs(t, client.Connect(context.Background(), "127.0.0.1:8080"), ErrNilDialer)
}

// TestClient_Disconnect_ClosesConnection checks that Disconnect sends QUIT and closes the socket, so the server
// sees the connection end, and that repeated or early calls are harmless.
func TestClient_Disconnect_ClosesConnection(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := connectFake(t, srv)

	assert.Equal(t, "2.11 (2.11.0) testserver", client.Version())

	require.NoError(t, client.Disconnect())

	assert.Nil(t, client.conn)
	assert.Empty(t, client.version, "a new connection must repeat the handshake")

	// The server closes its side only after reading everything sent before the client closed, including QUIT.
	require.Eventually(
		t,
		func() bool { return srv.OpenConns() == 0 },
		5*time.Second,
		10*time.Millisecond,
	)
	assert.Equal(t, "QUIT", lastCommand(srv))

	require.NoError(t, client.Disconnect(), "a second Disconnect is a no-op")

	neverConnected, err := NewClient(&net.Dialer{})
	require.NoError(t, err)
	require.NoError(t, neverConnected.Disconnect())
}

// TestClient_VersionContext checks the handshake against the fake server, its caching, and the no-connection error.
func TestClient_VersionContext(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := connectFake(t, srv)

	version, err := client.VersionContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2.11 (2.11.0) testserver", version)

	cached, err := client.VersionContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, version, cached)
	assert.Len(t, srv.Commands(), 1, "a cached version must not repeat HI")

	_, err = (&Client{}).VersionContext(context.Background())
	require.ErrorIs(t, err, ErrEmptyConn)
}

// TestClient_PingContext_FakeServer checks a ping against the fake server, which sends two PING commands.
func TestClient_PingContext_FakeServer(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := connectFake(t, srv)

	latency, err := client.PingContext(context.Background())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, latency, int64(0))
	assert.Len(t, srv.Commands(), 2)
}

// TestClient_PingContext_ContextEnds checks that cancellation and deadlines end a ping that the server never
// answers in time, promptly and with the context's error, instead of leaving a read blocked.
func TestClient_PingContext_ContextEnds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		context func() (context.Context, context.CancelFunc)
		wantErr error
		name    string
	}{
		{
			name: "cancelled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				time.AfterFunc(50*time.Millisecond, cancel)

				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testserver.NewTCPServer(t)
			srv.SetFaults(testserver.Faults{Delay: time.Second})

			client := connectFake(t, srv)

			ctx, cancel := tt.context()
			defer cancel()

			start := time.Now()
			_, err := client.PingContext(ctx)

			require.ErrorIs(t, err, tt.wantErr)
			require.NoError(
				t,
				client.Disconnect(),
				"a connection closed by the context must not linger",
			)
			assert.Less(
				t,
				time.Since(start),
				500*time.Millisecond,
				"the ping must not wait for the server",
			)
		})
	}
}

// TestClient_PingContext_ConnectionDropped checks that a server closing the connection mid-ping is reported as an
// I/O error rather than a context error.
func TestClient_PingContext_ConnectionDropped(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	srv.SetFaults(testserver.Faults{DropAfter: 2})

	client := connectFake(t, srv)

	_, err := client.PingContext(context.Background())

	require.ErrorIs(t, err, io.EOF)
	require.NotErrorIs(t, err, context.Canceled)
}

// TestClient_DeadlineCleared checks that an operation's deadline does not outlive it: a later call that sets no
// deadline of its own still works after the earlier deadline has passed.
func TestClient_DeadlineCleared(t *testing.T) {
	t.Parallel()

	tests := []struct {
		operation func(*Client, context.Context) error
		name      string
	}{
		{
			name: "ping",
			operation: func(client *Client, ctx context.Context) error {
				_, err := client.PingContext(ctx)

				return err
			},
		},
		{
			name: "handshake",
			operation: func(client *Client, ctx context.Context) error {
				_, err := client.VersionContext(ctx)

				return err
			},
		},
		{
			name: "cached handshake",
			operation: func(client *Client, ctx context.Context) error {
				client.version = "cached"

				// A deadline left by an earlier operation must not survive a cached handshake either.
				require.NoError(t, client.conn.SetDeadline(time.Now().Add(50*time.Millisecond)))

				_, err := client.VersionContext(ctx)

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testserver.NewTCPServer(t)
			client := connectFake(t, srv)

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			require.NoError(t, tt.operation(client, ctx))

			time.Sleep(100 * time.Millisecond)

			loss, err := client.PacketLoss()
			require.NoError(t, err, "the earlier deadline must not cut off later calls")
			assert.Equal(t, &PLoss{Sent: 0, Dup: 0, Max: 0}, loss)
		})
	}
}

// Test_watchContext checks who owns the outcome when an operation ends and its context is cancelled: completion
// keeps the connection, cancellation closes it and drops it from the client, and a replaced connection is kept.
func Test_watchContext(t *testing.T) {
	t.Parallel()

	// pipeClient returns a client on one end of an in-memory connection.
	pipeClient := func(t *testing.T) (*Client, net.Conn) {
		t.Helper()

		local, remote := net.Pipe()

		t.Cleanup(func() {
			_ = local.Close()
			_ = remote.Close()
		})

		return &Client{conn: local, reader: bufio.NewReader(local), version: "cached"}, local
	}

	// isClosed reports whether the connection has been closed.
	isClosed := func(conn net.Conn) bool {
		return conn.SetDeadline(time.Time{}) != nil
	}

	t.Run("completion keeps the connection", func(t *testing.T) {
		t.Parallel()

		client, conn := pipeClient(t)
		ctx, cancel := context.WithCancel(context.Background())

		stop := client.watchContext(ctx)
		stop()

		cancel()

		assert.Never(
			t,
			func() bool { return isClosed(conn) },
			100*time.Millisecond,
			10*time.Millisecond,
		)
		assert.Same(t, conn, client.conn)
		assert.Equal(t, "cached", client.version)
	})

	t.Run("cancellation closes and drops the connection", func(t *testing.T) {
		t.Parallel()

		client, conn := pipeClient(t)
		ctx, cancel := context.WithCancel(context.Background())

		stop := client.watchContext(ctx)

		cancel()

		require.Eventually(
			t,
			func() bool { return isClosed(conn) },
			5*time.Second,
			10*time.Millisecond,
		)

		stop()

		assert.Nil(t, client.conn)
		assert.Nil(t, client.reader)
		assert.Empty(t, client.version)
	})

	t.Run("a replaced connection is kept", func(t *testing.T) {
		t.Parallel()

		client, conn := pipeClient(t)
		replacement, _ := pipeClient(t)
		ctx, cancel := context.WithCancel(context.Background())

		stop := client.watchContext(ctx)

		cancel()

		require.Eventually(
			t,
			func() bool { return isClosed(conn) },
			5*time.Second,
			10*time.Millisecond,
		)

		client.conn = replacement.conn

		stop()

		assert.Same(t, replacement.conn, client.conn, "only the watched connection may be dropped")
	})
}

// sendLossDatagrams sends packet-loss datagrams for a client UUID to the fake server's UDP port.
func sendLossDatagrams(t *testing.T, srv *testserver.TCPServer, uuid string, orders ...int) {
	t.Helper()

	sender, err := NewPacketLossSender(uuid, &net.Dialer{})
	require.NoError(t, err)
	require.NoError(t, sender.Connect(context.Background(), srv.Addr()))

	defer func() { _ = sender.Close() }()

	for _, order := range orders {
		require.NoError(t, sender.Send(order))
	}
}

// TestClient_InitPacketLossContext_FakeServer checks that setup consumes the HELLO and OK replies, so the very
// first PLOSS request returns the current counts instead of a stale setup reply.
func TestClient_InitPacketLossContext_FakeServer(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := connectFake(t, srv)

	require.NoError(t, client.InitPacketLossContext(context.Background()))
	assert.Equal(t, "2.11 (2.11.0) testserver", client.version, "setup records the server version")

	sendLossDatagrams(t, srv, client.ID(), 0, 1, 1, 3)

	require.Eventually(
		t,
		func() bool { return srv.LossPackets(client.ID()) == 4 },
		5*time.Second,
		10*time.Millisecond,
	)

	loss, err := client.PacketLossContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, &PLoss{Sent: 4, Dup: 1, Max: 3}, loss)
}

// TestClient_InitPacketLossContext_Replies checks that setup rejects replies other than HELLO and OK.
func TestClient_InitPacketLossContext_Replies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		replies []string
	}{
		{name: "HI answered with something else", replies: []string{"ERROR\n"}},
		{
			name:    "INITPLOSS answered with something else",
			replies: []string{"HELLO 2.11\n", "ERROR\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			local, remote := net.Pipe()

			t.Cleanup(func() {
				_ = local.Close()
				_ = remote.Close()
			})

			go func() {
				reader := bufio.NewReader(remote)
				for _, reply := range tt.replies {
					_, err := reader.ReadString('\n')
					if err != nil {
						return
					}

					_, _ = io.WriteString(remote, reply)
				}
			}()

			client := &Client{id: "test-id", conn: local, reader: bufio.NewReader(local)}

			require.ErrorIs(
				t,
				client.InitPacketLossContext(context.Background()),
				ErrInvalidResponse,
			)
		})
	}

	t.Run("not connected", func(t *testing.T) {
		t.Parallel()

		require.ErrorIs(t, (&Client{}).InitPacketLossContext(context.Background()), ErrEmptyConn)
	})
}

// TestClient_PacketLossContext_Deadline checks that a stalled server cannot hold a sample past its deadline.
func TestClient_PacketLossContext_Deadline(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	srv.SetFaults(testserver.Faults{Delay: time.Second})

	client := connectFake(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.PacketLossContext(ctx)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 500*time.Millisecond)

	_, err = (&Client{}).PacketLossContext(context.Background())
	require.ErrorIs(t, err, ErrEmptyConn)
}

// expiredDeadline is a context whose deadline has passed while Err still reports nil, the window in which the
// socket's own timeout can fire first.
//
//nolint:containedctx // It wraps a context to override only its deadline.
type expiredDeadline struct {
	context.Context
}

// Deadline reports a deadline one second in the past.
func (expiredDeadline) Deadline() (time.Time, bool) {
	return time.Now().Add(-time.Second), true
}

// Test_contextError checks which errors are attributed to the context, including a socket timeout that arrives
// before the context reports its own deadline.
func Test_contextError(t *testing.T) {
	t.Parallel()

	timeout := &net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	// Parallel subtests run after this function returns, so cancel in cleanup rather than with defer.
	future, stop := context.WithTimeout(context.Background(), time.Hour)
	t.Cleanup(stop)

	tests := []struct {
		ctx     context.Context //nolint:containedctx // Each case needs its own context state.
		err     error
		wantErr []error
		name    string
	}{
		{
			name:    "cancelled context",
			ctx:     cancelled,
			err:     io.EOF,
			wantErr: []error{context.Canceled},
		},
		{
			name:    "socket timeout after the deadline",
			ctx:     expiredDeadline{context.Background()},
			err:     timeout,
			wantErr: []error{context.DeadlineExceeded, os.ErrDeadlineExceeded},
		},
		{
			name:    "socket timeout before the deadline",
			ctx:     future,
			err:     timeout,
			wantErr: []error{os.ErrDeadlineExceeded},
		},
		{
			name:    "other error without a deadline",
			ctx:     context.Background(),
			err:     io.EOF,
			wantErr: []error{io.EOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := contextError(tt.ctx, tt.err)

			for _, want := range tt.wantErr {
				require.ErrorIs(t, got, want)
			}

			if len(tt.wantErr) == 1 && tt.wantErr[0] == tt.err {
				assert.Same(t, tt.err, got, "unrelated errors pass through unchanged")
			}
		})
	}
}

// Test_watchContext_NoDone checks that watching a context that can never end starts nothing and stops cleanly.
func Test_watchContext_NoDone(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{conn: client1}

	stop := client.watchContext(context.Background())
	stop()

	require.NoError(t, client1.Close(), "the connection must still be open")
}

// TestClient_Download checks that a download hands the handler exactly the requested bytes, that bytes a handler
// leaves unread are discarded, and that the connection stays aligned for the next command either way.
func TestClient_Download(t *testing.T) {
	t.Parallel()

	const size = 100_000

	tests := []struct {
		name     string
		readUpTo int64
	}{
		{name: "handler reads everything", readUpTo: size + 1},
		{name: "handler stops early", readUpTo: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := connectFake(t, testserver.NewTCPServer(t))

			var read int64

			err := client.Download(context.Background(), size, func(stream io.Reader) error {
				var copyErr error

				read, copyErr = io.Copy(io.Discard, io.LimitReader(stream, tt.readUpTo))

				return copyErr
			})
			require.NoError(t, err)
			assert.Equal(t, min(tt.readUpTo, size), read)

			_, err = client.PingContext(context.Background())
			require.NoError(t, err, "the connection must be aligned for the next command")
		})
	}
}

// TestClient_DownloadErrors checks invalid requests, a truncated reply, and a failing handler.
func TestClient_DownloadErrors(t *testing.T) {
	t.Parallel()

	readAll := func(stream io.Reader) error {
		_, err := io.Copy(io.Discard, stream)

		return err
	}

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		client := connectFake(t, testserver.NewTCPServer(t))

		require.ErrorIs(t, client.Download(context.Background(), 0, readAll), ErrInvalidTransfer)
		require.ErrorIs(t, client.Download(context.Background(), 10, nil), ErrInvalidTransfer)
		require.ErrorIs(t, (&Client{}).Download(context.Background(), 10, readAll), ErrEmptyConn)
	})

	t.Run("truncated", func(t *testing.T) {
		t.Parallel()

		srv := testserver.NewTCPServer(t)
		srv.SetFaults(testserver.Faults{TruncateDownloads: true})

		err := connectFake(t, srv).Download(context.Background(), 100_000, readAll)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Contains(t, err.Error(), "after 50000 of 100000 bytes")
	})

	t.Run("handler error", func(t *testing.T) {
		t.Parallel()

		failure := errors.New("disk full")

		err := connectFake(t, testserver.NewTCPServer(t)).Download(
			context.Background(),
			1_000,
			func(io.Reader) error { return failure },
		)
		require.ErrorIs(t, err, failure)
	})
}

// TestClient_Upload checks that an upload returns the server's confirmed count and sends exactly the declared
// total, which the next command shows by getting its own reply.
func TestClient_Upload(t *testing.T) {
	t.Parallel()

	const size = 100_000

	srv := testserver.NewTCPServer(t)
	client := connectFake(t, srv)

	payload, err := UploadPayloadSize(size)
	require.NoError(t, err)

	confirmed, err := client.Upload(
		context.Background(),
		size,
		bytes.NewReader(make([]byte, payload)),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(size), confirmed)

	_, err = client.PingContext(context.Background())
	require.NoError(t, err, "the connection must be aligned for the next command")

	assert.True(t, slices.ContainsFunc(srv.Commands(), func(command testserver.Command) bool {
		return command.Line == "UPLOAD 100000 0"
	}))
}

// TestClient_UploadErrors checks invalid requests and a source that runs out before the payload is complete.
func TestClient_UploadErrors(t *testing.T) {
	t.Parallel()

	client := connectFake(t, testserver.NewTCPServer(t))
	source := bytes.NewReader(make([]byte, 100))

	for _, size := range []int64{-1, 0, 5} {
		_, err := client.Upload(context.Background(), size, source)
		require.ErrorIs(t, err, ErrInvalidTransfer, size)
	}

	_, err := client.Upload(context.Background(), 100, nil)
	require.ErrorIs(t, err, ErrInvalidTransfer)

	_, err = (&Client{}).Upload(context.Background(), 100, source)
	require.ErrorIs(t, err, ErrEmptyConn)

	_, err = client.Upload(context.Background(), 100_000, bytes.NewReader(make([]byte, 10)))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

// TestUploadPayloadSize checks the payload left after the command line, its newline, and the final newline.
func TestUploadPayloadSize(t *testing.T) {
	t.Parallel()

	payload, err := UploadPayloadSize(100)
	require.NoError(t, err)
	assert.Equal(t, int64(100-len("UPLOAD 100 0\n")-1), payload)

	payload, err = UploadPayloadSize(int64(len("UPLOAD 14 0\n") + 1))
	require.NoError(t, err)
	assert.Zero(
		t,
		payload,
		"a size that holds only the command and the newline has an empty payload",
	)

	for _, size := range []int64{-1, 0, 12} {
		_, err = UploadPayloadSize(size)
		require.ErrorIs(t, err, ErrInvalidTransfer, size)
	}
}

// Test_parsePong checks that the PONG parser accepts LF and CRLF endings and clocks of any length, and rejects
// other replies.
func Test_parsePong(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply string
		want  int64
	}{
		{name: "LF", reply: "PONG 1710000000000\n", want: 1710000000000},
		{name: "CRLF", reply: "PONG 1710000000000\r\n", want: 1710000000000},
		{name: "short clock", reply: "PONG 42\n", want: 42},
		{name: "long clock", reply: "PONG 17100000000000\n", want: 17100000000000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePong([]byte(tt.reply))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	for _, reply := range []string{"", "\n", "PONG \n", "PONG x\n", "PING 1710000000000\n", "HELLO 2.11\n"} {
		_, err := parsePong([]byte(reply))
		require.ErrorIs(t, err, ErrEchoData, reply)
	}
}

// Test_parseUploadReply checks the confirmation format and the bounds of its count.
func Test_parseUploadReply(t *testing.T) {
	t.Parallel()

	confirmed, err := parseUploadReply([]byte("OK 100 1710000000000\n"), 100)
	require.NoError(t, err)
	assert.Equal(t, int64(100), confirmed)

	for _, reply := range []string{"ERROR\n", "OK 100\n", "OK x 1\n", "OK -1 1\n", "OK 101 1\n", "PLOSS 1 2 3\n"} {
		_, err = parseUploadReply([]byte(reply), 100)
		require.ErrorIs(t, err, ErrInvalidResponse, reply)
	}
}

// TestClient_TransferCancelled checks that cancelling the context in the middle of a download or an upload ends it
// with the context's error and closes the connection.
func TestClient_TransferCancelled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		transfer func(ctx context.Context, cancel context.CancelFunc, client *Client) error
		name     string
	}{
		{
			name: "download",
			transfer: func(ctx context.Context, cancel context.CancelFunc, client *Client) error {
				return client.Download(ctx, 50_000_000, func(stream io.Reader) error {
					buffer := make([]byte, 1024)

					_, err := stream.Read(buffer)
					if err != nil {
						return err
					}

					cancel()

					_, err = io.Copy(io.Discard, stream)

					return err
				})
			},
		},
		{
			name: "upload",
			transfer: func(ctx context.Context, cancel context.CancelFunc, client *Client) error {
				_, err := client.Upload(ctx, 50_000_000, &cancellingReader{cancel: cancel})

				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testserver.NewTCPServer(t)
			client := connectFake(t, srv)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := tt.transfer(ctx, cancel, client)
			require.ErrorIs(t, err, context.Canceled)

			require.Eventually(
				t,
				func() bool { return srv.OpenConns() == 0 },
				5*time.Second,
				10*time.Millisecond,
			)
		})
	}
}

// cancellingReader supplies zero bytes and cancels a context after its first read.
type cancellingReader struct {
	cancel context.CancelFunc
	reads  int
}

// Read fills buffer with zeros, cancelling the context after the first call.
func (r *cancellingReader) Read(buffer []byte) (int, error) {
	r.reads++
	if r.reads == 2 {
		r.cancel()
	}

	clear(buffer)

	return len(buffer), nil
}

// TestClient_TransferFailureClosesConnection checks that a transfer failing partway closes the connection, so no
// later command reads leftover download bytes or leaves the server waiting for the rest of an upload.
func TestClient_TransferFailureClosesConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		transfer func(client *Client) error
		name     string
	}{
		{
			name: "download handler error",
			transfer: func(client *Client) error {
				return client.Download(context.Background(), 100_000, func(stream io.Reader) error {
					_, _ = stream.Read(make([]byte, 10))

					return errors.New("disk full")
				})
			},
		},
		{
			name: "upload source ends early",
			transfer: func(client *Client) error {
				_, err := client.Upload(
					context.Background(),
					100_000,
					bytes.NewReader(make([]byte, 10)),
				)

				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := testserver.NewTCPServer(t)
			client := connectFake(t, srv)

			require.Error(t, tt.transfer(client))

			// A reused connection would hang waiting for the server, so bound the check.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := client.PingContext(ctx)
			require.ErrorIs(t, err, ErrEmptyConn, "the failed connection must not be reused")

			require.Eventually(
				t,
				func() bool { return srv.OpenConns() == 0 },
				5*time.Second,
				10*time.Millisecond,
			)
		})
	}
}

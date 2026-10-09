// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUUID is the client UUID the packet-loss tests send.
const testUUID = "0f8fad5b-d9cb-469f-a165-70867728950e"

// session is a raw client connection to the fake TCP server.
type session struct {
	// conn is the TCP connection.
	conn net.Conn
	// reader buffers replies.
	reader *bufio.Reader
}

// dial opens a session to the fake TCP server and closes it when the test ends.
func dial(t *testing.T, srv *TCPServer) *session {
	t.Helper()

	dialer := net.Dialer{Timeout: 5 * time.Second}

	conn, err := dialer.DialContext(t.Context(), "tcp", srv.Addr())
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))

	return &session{conn: conn, reader: bufio.NewReader(conn)}
}

// send writes one command line.
func (s *session) send(t *testing.T, line string) {
	t.Helper()

	_, err := io.WriteString(s.conn, line+"\n")
	require.NoError(t, err)
}

// reply reads one reply line without its newline.
func (s *session) reply(t *testing.T) string {
	t.Helper()

	line, err := s.reader.ReadString('\n')
	require.NoError(t, err)

	return strings.TrimSuffix(line, "\n")
}

// ask sends a command and returns its reply.
func (s *session) ask(t *testing.T, line string) string {
	t.Helper()

	s.send(t, line)

	return s.reply(t)
}

// assertClosed checks that the server closed the session.
func (s *session) assertClosed(t *testing.T) {
	t.Helper()

	_, err := s.reader.ReadByte()
	assert.ErrorIs(t, err, io.EOF)
}

// sendLoss sends packet-loss datagrams with the given order numbers to the fake's UDP port.
func sendLoss(t *testing.T, srv *TCPServer, orders ...int) {
	t.Helper()

	var dialer net.Dialer

	conn, err := dialer.DialContext(t.Context(), "udp", srv.Addr())
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	for _, order := range orders {
		_, err := fmt.Fprintf(conn, "LOSS 1234567 %d %s", order, testUUID)
		require.NoError(t, err)
	}
}

// TestTCPServer_HelloAndPing checks the handshake reply and that PONG carries the server clock in a 19-byte line.
func TestTCPServer_HelloAndPing(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	assert.Equal(t, TCPHello, client.ask(t, "HI"))

	before := time.Now().UnixMilli()
	pong := client.ask(t, "PING "+strconv.FormatInt(time.Now().UnixNano(), 10))
	after := time.Now().UnixMilli()

	assert.Len(t, pong+"\n", 19, "clients require exactly 19 bytes including the newline")

	stamp, found := strings.CutPrefix(pong, "PONG ")
	require.True(t, found)

	millis, err := strconv.ParseInt(stamp, 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, millis, before)
	assert.LessOrEqual(t, millis, after)
}

// TestTCPServer_PacketLoss checks that HI <uuid> and INITPLOSS reply as on real servers, and that PLOSS reports the
// datagrams counted for the session's UUID.
func TestTCPServer_PacketLoss(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	assert.Equal(t, TCPHello, client.ask(t, "HI "+testUUID), "HI with a UUID still answers HELLO")
	assert.Equal(t, "OK", client.ask(t, "INITPLOSS"))
	assert.Equal(t, "PLOSS 0 0 0", client.ask(t, "PLOSS"))

	sendLoss(t, srv, 0, 1, 2, 3, 3, 5)

	require.Eventually(
		t,
		func() bool { return srv.LossPackets(testUUID) == 6 },
		5*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(
		t,
		"PLOSS 6 1 5",
		client.ask(t, "PLOSS"),
		"six received, one duplicate, highest order 5",
	)
}

// TestTCPServer_PacketLossIgnoresOtherSessions checks that counts are kept per UUID and that malformed datagrams
// are ignored.
func TestTCPServer_PacketLossIgnoresOtherSessions(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	client.ask(t, "HI 11111111-2222-3333-4444-555555555555")
	client.ask(t, "INITPLOSS")

	var dialer net.Dialer

	conn, err := dialer.DialContext(t.Context(), "udp", srv.Addr())
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	for _, datagram := range []string{"LOSS 1 x " + testUUID, "PING 1 2 3", "LOSS 1 2"} {
		_, err := io.WriteString(conn, datagram)
		require.NoError(t, err)
	}

	sendLoss(t, srv, 7)

	require.Eventually(
		t,
		func() bool { return srv.LossPackets(testUUID) == 1 },
		5*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, "PLOSS 0 0 0", client.ask(t, "PLOSS"), "another UUID's datagram must not count")
}

// TestTCPServer_LossDropEvery checks that the drop fault discards every Nth datagram.
func TestTCPServer_LossDropEvery(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	srv.SetFaults(Faults{LossDropEvery: 2})

	sendLoss(t, srv, 0, 1, 2, 3, 4, 5)

	require.Eventually(
		t,
		func() bool { return srv.LossPackets(testUUID) == 3 },
		5*time.Second,
		10*time.Millisecond,
	)

	client := dial(t, srv)
	client.ask(t, "HI "+testUUID)

	assert.Equal(t, "PLOSS 3 0 4", client.ask(t, "PLOSS"), "orders 1, 3, and 5 are dropped")
}

// TestTCPServer_Download checks that DOWNLOAD returns exactly the requested bytes and that the session continues.
func TestTCPServer_Download(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	client.send(t, "DOWNLOAD 100000")

	payload := make([]byte, 100000)
	_, err := io.ReadFull(client.reader, payload)
	require.NoError(t, err)

	assert.Equal(t, TCPHello, client.ask(t, "HI"), "the session continues after the payload")
}

// TestTCPServer_TruncateDownloads checks that the truncation fault closes the connection halfway through.
func TestTCPServer_TruncateDownloads(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	srv.SetFaults(Faults{TruncateDownloads: true})

	client := dial(t, srv)
	client.send(t, "DOWNLOAD 100000")

	data, err := io.ReadAll(client.reader)
	require.NoError(t, err)

	assert.Len(t, data, 50000)
}

// TestTCPServer_Upload checks that UPLOAD reads exactly the declared total and confirms it.
func TestTCPServer_Upload(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	payload := strings.Repeat("x", 5000) + "\n"

	// The declared total counts its own digits, so grow it until it is stable.
	total := len(payload)
	for range 3 {
		total = len(fmt.Sprintf("UPLOAD %d 0\n", total)) + len(payload)
	}

	command := fmt.Sprintf("UPLOAD %d 0", total)
	client.send(t, command)

	_, err := io.WriteString(client.conn, payload)
	require.NoError(t, err)

	fields := strings.Fields(client.reply(t))
	require.Len(t, fields, 3)
	assert.Equal(t, "OK", fields[0])
	assert.Equal(t, strconv.Itoa(total), fields[1])

	assert.Equal(t, TCPHello, client.ask(t, "HI"), "the payload must be consumed exactly")
}

// TestTCPServer_Errors checks that unknown and malformed commands answer ERROR without closing the session.
func TestTCPServer_Errors(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)

	for _, command := range []string{
		"GETIP",
		"   ",
		"DOWNLOAD",
		"DOWNLOAD many",
		"DOWNLOAD -1",
		"UPLOAD 10",
		"UPLOAD abc 0",
		"UPLOAD 3 0",
	} {
		assert.Equal(t, "ERROR", client.ask(t, command), command)
	}

	assert.Equal(t, TCPHello, client.ask(t, "HI"))
}

// TestTCPServer_Quit checks that QUIT closes the connection, also when sent without a newline.
func TestTCPServer_Quit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "with newline", raw: "QUIT\n"},
		{name: "without newline", raw: "QUIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := NewTCPServer(t)
			client := dial(t, srv)

			_, err := io.WriteString(client.conn, tt.raw)
			require.NoError(t, err)

			if tcp, ok := client.conn.(*net.TCPConn); ok && !strings.HasSuffix(tt.raw, "\n") {
				require.NoError(t, tcp.CloseWrite())
			}

			client.assertClosed(t)

			require.Eventually(
				t,
				func() bool { return srv.OpenConns() == 0 },
				5*time.Second,
				10*time.Millisecond,
			)
		})
	}
}

// TestTCPServer_DropAfter checks that the drop fault closes the connection instead of answering the Nth command.
func TestTCPServer_DropAfter(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	srv.SetFaults(Faults{DropAfter: 2})

	client := dial(t, srv)

	assert.Equal(t, TCPHello, client.ask(t, "HI"))

	client.send(t, "PING 1")
	client.assertClosed(t)
}

// TestTCPServer_Delay checks that the delay fault holds back each reply.
func TestTCPServer_Delay(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	srv.SetFaults(Faults{Delay: 50 * time.Millisecond})

	client := dial(t, srv)

	start := time.Now()

	assert.Equal(t, TCPHello, client.ask(t, "HI"))
	assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond)
}

// TestTCPServer_Bookkeeping checks the command log, the accepted count, and that client closes are noticed.
func TestTCPServer_Bookkeeping(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)

	first := dial(t, srv)
	first.ask(t, "HI")

	second := dial(t, srv)
	second.ask(t, "PING 1")

	assert.Equal(t, 2, srv.Accepted())
	assert.Equal(t, []Command{{Line: "HI", Conn: 0}, {Line: "PING 1", Conn: 1}}, srv.Commands())
	assert.Equal(t, 2, srv.OpenConns())

	require.NoError(t, first.conn.Close())
	require.NoError(t, second.conn.Close())

	require.Eventually(
		t,
		func() bool { return srv.OpenConns() == 0 },
		5*time.Second,
		10*time.Millisecond,
	)
}

// TestTCPServer_CloseEndsSessions checks that stopping the server closes open sessions instead of leaving clients
// blocked.
func TestTCPServer_CloseEndsSessions(t *testing.T) {
	t.Parallel()

	srv := NewTCPServer(t)
	client := dial(t, srv)
	client.ask(t, "HI")

	srv.close()

	_, err := client.reader.ReadByte()
	require.Error(t, err)
	assert.True(t, errors.Is(err, io.EOF) || isReset(err), "unexpected error: %v", err)
}

// TestTCPServer_CloseWhileAccepting checks that stopping the server while clients keep connecting closes every
// accepted connection and returns, instead of serving a connection accepted during shutdown and waiting on it.
func TestTCPServer_CloseWhileAccepting(t *testing.T) {
	t.Parallel()

	for range 100 {
		srv := NewTCPServer(t)
		stop := make(chan struct{})

		var (
			conns []net.Conn
			mu    sync.Mutex
			wg    sync.WaitGroup
		)

		track := func(conn net.Conn) {
			mu.Lock()
			defer mu.Unlock()

			conns = append(conns, conn)
		}

		for range 8 {
			wg.Go(func() {
				dialer := net.Dialer{Timeout: time.Second}

				for {
					select {
					case <-stop:
						return
					default:
					}

					conn, err := dialer.DialContext(t.Context(), "tcp", srv.Addr())
					if err != nil {
						return
					}

					track(conn)
				}
			})
		}

		time.Sleep(2 * time.Millisecond)

		closed := make(chan struct{})

		go func() {
			srv.close()
			close(closed)
		}()

		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "close did not return while clients were connecting")
		}

		close(stop)
		wg.Wait()

		for _, conn := range conns {
			_ = conn.Close()
		}
	}
}

// errWSAConnReset is the Windows Sockets connection reset error, which Go reports on Windows instead of ECONNRESET.
const errWSAConnReset = syscall.Errno(10054)

// isReset reports whether err is a connection reset, which some platforms report instead of EOF.
func isReset(err error) bool {
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, errWSAConnReset)
}

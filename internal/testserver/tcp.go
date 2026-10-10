// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TCPServer is a fake of a speedtest server's plain-text TCP protocol and its UDP packet-loss listener.
//
// Both listen on the same loopback port, as on a real server, where clients send packet-loss datagrams to the
// host and port of the TCP control connection. Replies match those of real servers: "HI" and "HI <uuid>" answer
// HELLO, "INITPLOSS" answers OK, "PLOSS" answers the counts for the connection's UUID, and "PING" answers PONG with
// the server's clock in milliseconds.
type TCPServer struct {
	// listener accepts TCP control connections.
	listener net.Listener
	// packets receives UDP packet-loss datagrams on the same port.
	packets net.PacketConn
	// conns holds the open connections, so Close can end them.
	conns map[net.Conn]struct{}
	// loss counts received datagrams per order number, keyed by client UUID.
	loss map[string]map[int]int
	// commands records every command line received, in order.
	commands []Command
	// faults are the failures to inject.
	faults Faults
	// syncWaiters maps a pending SyncPackets token to the channel closed when its marker is handled.
	syncWaiters map[int]chan struct{}
	// accepted counts every accepted connection.
	accepted int
	// syncSeq is the last token issued by SyncPackets.
	syncSeq int
	// wg tracks the accept, UDP, and connection goroutines.
	wg sync.WaitGroup
	// mu guards conns, loss, commands, faults, accepted, closed, syncWaiters, and syncSeq.
	mu sync.Mutex
	// closed is set when close starts, so connections accepted afterwards are closed instead of served.
	closed bool
}

// Command is a command line the fake TCP server received.
type Command struct {
	// Line is the command without its trailing newline.
	Line string
	// Conn is the index of the connection it arrived on, counting from 0 in accept order.
	Conn int
}

// Faults are failures the fake TCP server injects into later commands.
type Faults struct {
	// Delay is waited before each reply.
	Delay time.Duration
	// DropAfter closes a connection instead of answering its Nth command. Zero never drops.
	DropAfter int
	// LossDropEvery discards every Nth packet-loss datagram, counting from the first. Zero discards none.
	LossDropEvery int
	// TruncateDownloads closes the connection halfway through every DOWNLOAD reply.
	TruncateDownloads bool
}

// TCPHello is the HELLO reply the fake TCP server sends, in the format of real servers.
const TCPHello = "HELLO 2.11 (2.11.0) testserver"

// listenAttempts bounds the retries when the UDP port matching a new TCP port is already taken.
const listenAttempts = 10

// Protocol command names.
const (
	// cmdHi starts a session, optionally with the client's UUID.
	cmdHi = "HI"
	// cmdPing asks for the server's clock.
	cmdPing = "PING"
	// cmdInitLoss starts packet-loss counting for the session's UUID.
	cmdInitLoss = "INITPLOSS"
	// cmdLoss asks for the packet-loss counts of the session's UUID.
	cmdLoss = "PLOSS"
	// cmdDownload asks for a number of bytes.
	cmdDownload = "DOWNLOAD"
	// cmdUpload announces an upload and its total size.
	cmdUpload = "UPLOAD"
	// cmdQuit ends the session.
	cmdQuit = "QUIT"
	// lossDatagram starts every UDP packet-loss datagram.
	lossDatagram = "LOSS"
	// syncDatagram starts the marker datagrams sent by SyncPackets.
	syncDatagram = "SYNC"
)

// syncTimeout bounds how long SyncPackets waits for its marker.
const syncTimeout = 5 * time.Second

// errListen reports that no port was free for both TCP and UDP.
var errListen = errors.New("no loopback port free for both TCP and UDP")

// NewTCPServer starts a fake TCP server and its UDP listener on one loopback port, and stops them when the test
// ends.
//
// Parameters:
//   - tb: the test or benchmark that owns the server.
//
// Returns:
//   - *TCPServer: the running fake.
func NewTCPServer(tb testing.TB) *TCPServer {
	tb.Helper()

	listener, packets, err := listenPair()
	if err != nil {
		tb.Fatalf("start fake TCP server: %v", err)
	}

	srv := &TCPServer{
		listener:    listener,
		packets:     packets,
		conns:       map[net.Conn]struct{}{},
		loss:        map[string]map[int]int{},
		syncWaiters: map[int]chan struct{}{},
	}

	srv.wg.Add(2)

	go srv.acceptLoop()
	go srv.packetLoop()

	tb.Cleanup(srv.close)

	return srv
}

// Addr returns the host and port that clients dial for both TCP and UDP.
//
// Returns:
//   - string: the loopback host:port.
func (srv *TCPServer) Addr() string {
	return srv.listener.Addr().String()
}

// SetFaults replaces the failures injected into later commands and datagrams.
//
// Parameters:
//   - faults: the failures to inject.
func (srv *TCPServer) SetFaults(faults Faults) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.faults = faults
}

// Commands returns a copy of every command received so far, in order.
//
// Returns:
//   - []Command: the recorded commands.
func (srv *TCPServer) Commands() []Command {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return slices.Clone(srv.commands)
}

// Accepted returns the number of connections accepted so far.
//
// Returns:
//   - int: the accepted connection count.
func (srv *TCPServer) Accepted() int {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return srv.accepted
}

// OpenConns returns the number of connections that are still open on the server side.
//
// A connection closes when the client closes it, sends QUIT, or a fault drops it. The server notices a client close
// only when its next read fails, so callers should poll, for example with require.Eventually.
//
// Returns:
//   - int: the open connection count.
func (srv *TCPServer) OpenConns() int {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return len(srv.conns)
}

// LossPackets returns the number of packet-loss datagrams counted for a client UUID, including duplicates.
//
// Parameters:
//   - uuid: the client UUID sent in "HI <uuid>" and in each datagram.
//
// Returns:
//   - int: the number of counted datagrams.
func (srv *TCPServer) LossPackets(uuid string) int {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	var total int
	for _, count := range srv.loss[uuid] {
		total += count
	}

	return total
}

// SyncPackets waits until the packet loop has handled every datagram that reached the UDP port before the call.
//
// It sends a marker datagram and waits for the packet loop to handle it. On loopback a datagram is queued at the
// receiver when its send returns, so the marker queues behind every datagram already sent, and handling it means
// those have been counted. Call it after the senders have returned and before reading [TCPServer.LossPackets].
// Markers are not counted and do not advance [Faults.LossDropEvery].
//
// Parameters:
//   - tb: the test or benchmark, failed when the marker cannot be sent or is not handled in time.
func (srv *TCPServer) SyncPackets(tb testing.TB) {
	tb.Helper()

	handled := make(chan struct{})

	srv.mu.Lock()
	srv.syncSeq++
	token := srv.syncSeq
	srv.syncWaiters[token] = handled
	srv.mu.Unlock()

	var dialer net.Dialer

	conn, err := dialer.DialContext(context.Background(), "udp", srv.Addr())
	if err != nil {
		tb.Fatalf("dial fake UDP port: %v", err)
	}

	defer func() { _ = conn.Close() }()

	_, err = fmt.Fprintf(conn, "%s %d", syncDatagram, token)
	if err != nil {
		tb.Fatalf("send sync marker: %v", err)
	}

	select {
	case <-handled:
	case <-time.After(syncTimeout):
		tb.Fatalf(
			"fake UDP packet loop did not handle sync marker %d within %s",
			token,
			syncTimeout,
		)
	}
}

// acceptLoop accepts control connections until the listener closes.
func (srv *TCPServer) acceptLoop() {
	defer srv.wg.Done()

	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			return
		}

		srv.mu.Lock()

		if srv.closed {
			srv.mu.Unlock()

			_ = conn.Close()

			continue
		}

		index := srv.accepted
		srv.accepted++
		srv.conns[conn] = struct{}{}
		srv.wg.Add(1)
		srv.mu.Unlock()

		go srv.serveConn(conn, index)
	}
}

// serveConn answers the commands of one control connection until it closes.
//
// Parameters:
//   - conn: the connection.
//   - index: the connection's accept order, recorded with each command.
func (srv *TCPServer) serveConn(conn net.Conn, index int) {
	defer srv.wg.Done()
	defer srv.closeConn(conn)

	reader := bufio.NewReader(conn)

	var (
		uuid    string
		handled int
	)

	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if line != "" {
			handled++

			srv.mu.Lock()
			srv.commands = append(srv.commands, Command{Line: line, Conn: index})
			faults := srv.faults
			srv.mu.Unlock()

			if faults.DropAfter > 0 && handled >= faults.DropAfter {
				return
			}

			time.Sleep(faults.Delay)

			if !srv.answer(conn, reader, line, &uuid, faults) {
				return
			}
		}

		if readErr != nil {
			return
		}
	}
}

// answer writes the reply to one command.
//
// Parameters:
//   - conn: the connection to reply on.
//   - reader: the connection's buffered reader, which supplies upload payloads.
//   - line: the command line without its newline.
//   - uuid: the session's client UUID, set by "HI <uuid>".
//   - faults: the failures to inject.
//
// Returns:
//   - bool: false when the connection must close.
func (srv *TCPServer) answer(
	conn net.Conn,
	reader *bufio.Reader,
	line string,
	uuid *string,
	faults Faults,
) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return writeLine(conn, "ERROR")
	}

	switch fields[0] {
	case cmdHi:
		if len(fields) > 1 {
			*uuid = fields[1]
		}

		return writeLine(conn, TCPHello)
	case cmdPing:
		return writeLine(conn, fmt.Sprintf("PONG %013d", time.Now().UnixMilli()))
	case cmdInitLoss:
		srv.mu.Lock()
		if _, ok := srv.loss[*uuid]; !ok {
			srv.loss[*uuid] = map[int]int{}
		}
		srv.mu.Unlock()

		return writeLine(conn, "OK")
	case cmdLoss:
		sent, dup, highest := srv.lossCounts(*uuid)

		return writeLine(conn, fmt.Sprintf("PLOSS %d %d %d", sent, dup, highest))
	case cmdDownload:
		return writeDownload(conn, fields, faults.TruncateDownloads)
	case cmdUpload:
		return readUpload(conn, reader, line, fields)
	case cmdQuit:
		return false
	default:
		return writeLine(conn, "ERROR")
	}
}

// lossCounts summarizes the datagrams counted for a client UUID.
//
// Parameters:
//   - uuid: the client UUID.
//
// Returns:
//   - sent: the number of datagrams received, including duplicates.
//   - dup: the number of duplicate datagrams.
//   - highest: the highest order number received, or 0 when none arrived.
//
//nolint:nonamedreturns // Same-type returns need names.
func (srv *TCPServer) lossCounts(uuid string) (sent, dup, highest int) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	for order, count := range srv.loss[uuid] {
		sent += count
		dup += count - 1
		highest = max(highest, order)
	}

	return sent, dup, highest
}

// packetLoop counts packet-loss datagrams until the UDP listener closes.
//
// A datagram has the form "LOSS <nonce> <order> <uuid>". Malformed datagrams are ignored.
func (srv *TCPServer) packetLoop() {
	defer srv.wg.Done()

	buf := make([]byte, 1500)

	var received int

	for {
		n, _, err := srv.packets.ReadFrom(buf)
		if err != nil {
			return
		}

		fields := strings.Fields(string(buf[:n]))

		if len(fields) == 2 && fields[0] == syncDatagram {
			srv.handleSync(fields[1])

			continue
		}

		if len(fields) != 4 || fields[0] != lossDatagram {
			continue
		}

		order, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		received++

		srv.mu.Lock()

		if every := srv.faults.LossDropEvery; every > 0 && received%every == 0 {
			srv.mu.Unlock()

			continue
		}

		counts, ok := srv.loss[fields[3]]
		if !ok {
			counts = map[int]int{}
			srv.loss[fields[3]] = counts
		}

		counts[order]++
		srv.mu.Unlock()
	}
}

// handleSync releases the SyncPackets call waiting for a marker token.
//
// Parameters:
//   - token: the marker's token, as sent.
func (srv *TCPServer) handleSync(token string) {
	value, err := strconv.Atoi(token)
	if err != nil {
		return
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if handled, ok := srv.syncWaiters[value]; ok {
		close(handled)
		delete(srv.syncWaiters, value)
	}
}

// closeConn closes a connection and forgets it.
//
// Parameters:
//   - conn: the connection.
func (srv *TCPServer) closeConn(conn net.Conn) {
	_ = conn.Close()

	srv.mu.Lock()
	delete(srv.conns, conn)
	srv.mu.Unlock()
}

// close stops both listeners, closes every open connection, and waits for all goroutines to finish.
//
// It marks the server closed under the same lock that guards connection registration, so a connection accepted
// while close runs is either registered before the sweep below and closed by it, or closed by acceptLoop.
func (srv *TCPServer) close() {
	srv.mu.Lock()
	srv.closed = true
	srv.mu.Unlock()

	_ = srv.listener.Close()
	_ = srv.packets.Close()

	srv.mu.Lock()
	for conn := range srv.conns {
		_ = conn.Close()
	}
	srv.mu.Unlock()

	srv.wg.Wait()
}

// listenPair opens a TCP listener and a UDP listener on the same loopback port.
//
// Returns:
//   - net.Listener: the TCP listener.
//   - net.PacketConn: the UDP listener.
//   - error: errListen when no attempt found a port free for both.
func listenPair() (net.Listener, net.PacketConn, error) {
	var config net.ListenConfig

	for range listenAttempts {
		listener, err := config.Listen(context.Background(), "tcp", "127.0.0.1:0")
		if err != nil {
			return nil, nil, fmt.Errorf("listen TCP: %w", err)
		}

		packets, err := config.ListenPacket(context.Background(), "udp", listener.Addr().String())
		if err == nil {
			return listener, packets, nil
		}

		_ = listener.Close()
	}

	return nil, nil, errListen
}

// writeLine writes one reply line.
//
// Parameters:
//   - conn: the connection.
//   - line: the reply without its newline.
//
// Returns:
//   - bool: false when the write failed.
func writeLine(conn net.Conn, line string) bool {
	_, err := io.WriteString(conn, line+"\n")

	return err == nil
}

// writeDownload answers "DOWNLOAD <bytes>" with that many bytes.
//
// Parameters:
//   - conn: the connection.
//   - fields: the command's fields.
//   - truncate: whether to close halfway through.
//
// Returns:
//   - bool: false when the connection must close.
func writeDownload(conn net.Conn, fields []string, truncate bool) bool {
	if len(fields) != 2 {
		return writeLine(conn, "ERROR")
	}

	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || size < 0 {
		return writeLine(conn, "ERROR")
	}

	if truncate {
		size /= 2
	}

	for remaining := size; remaining > 0; {
		chunk := downloadChunk[:min(remaining, int64(len(downloadChunk)))]

		written, err := conn.Write(chunk)
		if err != nil {
			return false
		}

		remaining -= int64(written)
	}

	return !truncate
}

// readUpload answers "UPLOAD <total> 0" by reading the rest of the declared total and confirming it.
//
// The declared total counts the command line and its newline, the payload, and the payload's final newline.
//
// Parameters:
//   - conn: the connection.
//   - reader: the buffered reader that supplies the payload.
//   - line: the command line without its newline.
//   - fields: the command's fields.
//
// Returns:
//   - bool: false when the connection must close.
func readUpload(conn net.Conn, reader *bufio.Reader, line string, fields []string) bool {
	if len(fields) != 3 {
		return writeLine(conn, "ERROR")
	}

	total, err := strconv.ParseInt(fields[1], 10, 64)
	payload := total - int64(len(line)+1)

	if err != nil || payload < 0 {
		return writeLine(conn, "ERROR")
	}

	_, err = io.CopyN(io.Discard, reader, payload)
	if err != nil {
		return false
	}

	return writeLine(conn, fmt.Sprintf("OK %d %d", total, time.Now().UnixMilli()))
}

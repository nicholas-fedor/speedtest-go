// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// Command words of the TCP protocol, in ASCII.
var (
	// pingPrefix is "PING ", which starts a latency probe carrying the client's clock.
	pingPrefix = []byte{0x50, 0x49, 0x4e, 0x47, 0x20}
	// initPacket is "INITPLOSS", which starts the server's packet-loss count for the client's UUID.
	initPacket = []byte{0x49, 0x4e, 0x49, 0x54, 0x50, 0x4c, 0x4f, 0x53, 0x53}
	// packetLoss is "PLOSS", which asks for the packet-loss counts and starts the server's reply.
	packetLoss = []byte{0x50, 0x4c, 0x4f, 0x53, 0x53}
	// hiFormat is "HI", the handshake, which the server answers with its version.
	hiFormat = []byte{0x48, 0x49}
	// quitFormat is "QUIT", which ends the session.
	quitFormat = []byte{0x51, 0x55, 0x49, 0x54}
)

// Errors returned by the transport clients.
var (
	// ErrEchoData is returned when the echo data is incorrect.
	ErrEchoData = errors.New("incorrect echo data")
	// ErrEmptyConn is returned when the connection is empty.
	ErrEmptyConn = errors.New("empty conn")
	// ErrUninitializedPacketLossInst is returned when the packet loss instance is not initialized.
	ErrUninitializedPacketLossInst = errors.New("uninitialized packet loss inst")
	// ErrInvalidPacketLossResponse is returned when the packet loss response is invalid.
	ErrInvalidPacketLossResponse = errors.New("invalid packet loss response")
	// ErrInvalidResponse is returned when the server's reply does not follow the protocol.
	ErrInvalidResponse = errors.New("invalid TCP speedtest response")
	// ErrNilDialer is returned when a client is created without a dialer and asked to connect.
	ErrNilDialer = errors.New("transport client has no dialer")
	// ErrInvalidTransfer is returned for a download or upload with a size, handler, or source it cannot use.
	ErrInvalidTransfer = errors.New("invalid transfer request")
)

// Server replies checked during the handshake and packet-loss setup.
var (
	// helloPrefix starts the server's reply to HI and to "HI <uuid>".
	helloPrefix = []byte("HELLO ")
	// okReply is the server's reply to INITPLOSS, and the first field of its reply to an upload.
	okReply = []byte("OK")
	// pongPrefix starts the server's reply to PING, which carries its clock in milliseconds.
	pongPrefix = []byte("PONG ")
)

// Transfer command prefixes.
var (
	// downloadPrefix asks the server for a number of bytes.
	downloadPrefix = []byte("DOWNLOAD ")
	// uploadPrefix announces the total size of an upload.
	uploadPrefix = []byte("UPLOAD ")
)

// pingFormat builds a PING command for a local timestamp, copying the shared prefix so callers never alias it.
//
// Parameters:
//   - locTime: the client's clock in nanoseconds.
//
// Returns:
//   - []byte: the command without its newline.
func pingFormat(locTime int64) []byte {
	return strconv.AppendInt(bytes.Clone(pingPrefix), locTime, 10)
}

// downloadFormat builds a "DOWNLOAD <bytes>" command.
//
// Parameters:
//   - size: the number of bytes to request.
//
// Returns:
//   - []byte: the command without its newline.
func downloadFormat(size int64) []byte {
	return strconv.AppendInt(bytes.Clone(downloadPrefix), size, 10)
}

// uploadFormat builds an "UPLOAD <total> 0" command.
//
// Parameters:
//   - size: the declared total size of the upload.
//
// Returns:
//   - []byte: the command without its newline.
func uploadFormat(size int64) []byte {
	return append(strconv.AppendInt(bytes.Clone(uploadPrefix), size, 10), ' ', '0')
}

// UploadPayloadSize returns how many data bytes an upload of a declared total size carries.
//
// The total counts the command line and its newline, the payload, and the newline that ends the payload, so the
// payload is the total less those three parts.
//
// Parameters:
//   - size: the declared total size.
//
// Returns:
//   - int64: the payload size.
//   - error: an error wrapping [ErrInvalidTransfer] when size cannot hold the command and the final newline.
func UploadPayloadSize(size int64) (int64, error) {
	if size <= 0 {
		return 0, fmt.Errorf("%w: upload size %d", ErrInvalidTransfer, size)
	}

	payload := size - int64(len(uploadFormat(size))+1) - 1
	if payload < 0 {
		return 0, fmt.Errorf(
			"%w: upload size %d is smaller than its command",
			ErrInvalidTransfer,
			size,
		)
	}

	return payload, nil
}

// Client represents a TCP client for speedtest operations.
type Client struct {
	// conn is the connection to the server, or nil when not connected.
	conn net.Conn

	// dialer opens the connection.
	dialer *net.Dialer

	// reader buffers the replies read from conn.
	reader *bufio.Reader
	// id is the UUID the server counts packet loss under.
	id string
	// host is the server's host:port.
	host string
	// version caches the server version from its HELLO reply, or is empty before the handshake.
	version string
}

// NewClient creates a new TCP client with the given dialer.
//
// Parameters:
//   - dialer: opens the connection. A nil dialer makes [Client.Connect] fail with [ErrNilDialer].
//
// Returns:
//   - *Client: the client, with a new UUID and no connection.
//   - error: the error from generating the UUID.
func NewClient(dialer *net.Dialer) (*Client, error) {
	uuid, err := generateUUID()
	if err != nil {
		return nil, err
	}

	return &Client{
		id:     uuid,
		dialer: dialer,
	}, nil
}

// ID returns the client ID.
//
// Returns:
//   - string: the UUID the server counts packet loss under.
func (client *Client) ID() string {
	return client.id
}

// Connect establishes a connection to the specified host.
//
// Parameters:
//   - ctx: cancellation for the dial.
//   - host: the server's host:port.
//
// Returns:
//   - error: [ErrNilDialer] when the client has no dialer, or the dial error.
func (client *Client) Connect(ctx context.Context, host string) error {
	if client.dialer == nil {
		return ErrNilDialer
	}

	client.host = host

	conn, err := client.dialer.DialContext(ctx, "tcp", client.host)
	if err != nil {
		return fmt.Errorf("failed to dial TCP: %w", err)
	}

	client.conn = conn
	client.reader = bufio.NewReader(client.conn)

	return nil
}

// Disconnect ends the session with QUIT and closes the connection.
//
// It is safe to call on a client that never connected or already disconnected. QUIT is sent on a best-effort
// basis, because the server may already have closed its side.
//
// Returns:
//   - error: the error from closing the connection, if any.
func (client *Client) Disconnect() error {
	if client.conn == nil {
		return nil
	}

	_ = client.writeAll(append(bytes.Clone(quitFormat), '\n'))

	err := client.conn.Close()

	client.conn = nil
	client.reader = nil
	client.version = ""

	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

// Write sends one command line, adding its newline.
//
// Parameters:
//   - data: the command without its newline.
//
// Returns:
//   - error: [ErrEmptyConn] when not connected, or the write error.
func (client *Client) Write(data []byte) error {
	if client.conn == nil {
		return ErrEmptyConn
	}

	return client.writeAll(append(bytes.Clone(data), '\n'))
}

// Read reads one reply line from the server.
//
// Returns:
//   - []byte: the line, including its newline.
//   - error: [ErrEmptyConn] when not connected, or the read error.
func (client *Client) Read() ([]byte, error) {
	if client.conn == nil {
		return nil, ErrEmptyConn
	}

	data, err := client.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read from connection: %w", err)
	}

	return data, nil
}

// Version returns the server's version string, or "unknown" when the handshake fails.
//
// Returns:
//   - string: the version from the server's HELLO reply.
func (client *Client) Version() string {
	version, err := client.VersionContext(context.Background())
	if err != nil {
		return "unknown"
	}

	return version
}

// VersionContext performs the HI handshake, observing ctx, and caches the server's version.
//
// Parameters:
//   - ctx: cancellation and deadline for the handshake.
//
// Returns:
//   - string: the version from the server's HELLO reply.
//   - error: [ErrEmptyConn] when not connected, [ErrInvalidResponse] when the reply is not HELLO, or the I/O error.
func (client *Client) VersionContext(ctx context.Context) (string, error) {
	if client.conn == nil {
		return "", ErrEmptyConn
	}

	defer client.clearDeadline()

	if client.version != "" {
		return client.version, nil
	}

	stop := client.watchContext(ctx)
	defer stop()

	err := client.setDeadline(ctx)
	if err != nil {
		return "", err
	}

	err = client.Write(hiFormat)
	if err != nil {
		return "", contextError(ctx, err)
	}

	message, err := client.Read()
	if err != nil {
		return "", contextError(ctx, err)
	}

	version, err := parseHello(message)
	if err != nil {
		return "", err
	}

	client.version = version

	return client.version, nil
}

// PingContext measures the latency (RTT) between client and server, observing ctx.
//
// It uses the 2RTT method to obtain three RTT results in less time (t2-t0, t4-t2, t3-t1), and gives the delay
// measured by the server a lower weight:
//
//	latency = 0.4 * (t2 - t0) + 0.4 * (t4 - t2) + 0.2 * (t3 - t1)
//
// The exchange runs on the calling goroutine. The context's deadline is applied to the connection for the exchange
// only, and cancelling the context closes the connection, so a blocked read returns instead of leaking. A
// connection closed by cancellation is dropped from the client.
//
// Parameters:
//   - ctx: cancellation and deadline for the exchange.
//
// Returns:
//   - int64: the weighted latency in nanoseconds.
//   - error: [ErrEmptyConn] when not connected, [ErrEchoData] for a malformed reply, the context's error when it
//     ends the exchange, or the I/O error.
func (client *Client) PingContext(ctx context.Context) (int64, error) {
	if client.conn == nil {
		return 0, ErrEmptyConn
	}

	defer client.clearDeadline()

	stop := client.watchContext(ctx)
	defer stop()

	err := client.setDeadline(ctx)
	if err != nil {
		return 0, err
	}

	var accumulatedLatency, firstReceivedByServer int64 // firstReceivedByServer is t1

	for i := range 2 {
		t0 := time.Now().UnixNano()

		err := client.Write(pingFormat(t0))
		if err != nil {
			return 0, contextError(ctx, err)
		}

		data, err := client.Read()
		t2 := time.Now().UnixNano()

		if err != nil {
			return 0, contextError(ctx, err)
		}

		tx, err := parsePong(data)
		if err != nil {
			return 0, err
		}

		accumulatedLatency += (t2 - t0) * 4 / 10 // 0.4

		if i == 0 {
			firstReceivedByServer = tx
		} else {
			// append server-side latency result
			accumulatedLatency += (tx - firstReceivedByServer) * 1000 * 1000 * 2 / 10 // 0.2
		}
	}

	return accumulatedLatency, nil
}

// InitPacketLoss registers the client's UUID for packet-loss counting. See [Client.InitPacketLossContext].
//
// Returns:
//   - error: the error from [Client.InitPacketLossContext].
func (client *Client) InitPacketLoss() error {
	return client.InitPacketLossContext(context.Background())
}

// InitPacketLossContext registers the client's UUID for packet-loss counting, observing ctx.
//
// It sends "HI <uuid>" and INITPLOSS and reads both replies, HELLO and OK. Reading them keeps later PLOSS replies
// aligned with their requests. The HELLO reply also records the server version.
//
// Parameters:
//   - ctx: cancellation and deadline for the setup.
//
// Returns:
//   - error: [ErrEmptyConn] when not connected, [ErrInvalidResponse] for an unexpected reply, the context's error
//     when it ends the setup, or the I/O error.
func (client *Client) InitPacketLossContext(ctx context.Context) error {
	if client.conn == nil {
		return ErrEmptyConn
	}

	defer client.clearDeadline()

	stop := client.watchContext(ctx)
	defer stop()

	err := client.setDeadline(ctx)
	if err != nil {
		return err
	}

	hello, err := client.exchange(ctx, append(append(bytes.Clone(hiFormat), ' '), client.id...))
	if err != nil {
		return err
	}

	version, err := parseHello(hello)
	if err != nil {
		return err
	}

	client.version = version

	reply, err := client.exchange(ctx, initPacket)
	if err != nil {
		return err
	}

	if !bytes.Equal(bytes.TrimRight(reply, "\r\n"), okReply) {
		return fmt.Errorf("%w: %q", ErrInvalidResponse, reply)
	}

	return nil
}

// PLoss holds the packet-loss counts the server reports for a client's UUID.
type PLoss struct {
	// Sent is the number of datagrams the server received.
	Sent int
	// Dup is the number of duplicate datagrams the server received.
	Dup int
	// Max is the highest datagram index the server received.
	Max int
}

// PacketLoss retrieves the packet-loss counts from the server. See [Client.PacketLossContext].
//
// Returns:
//   - *PLoss: the counts.
//   - error: the error from [Client.PacketLossContext].
func (client *Client) PacketLoss() (*PLoss, error) {
	return client.PacketLossContext(context.Background())
}

// PacketLossContext retrieves the packet-loss counts for the client's UUID, observing ctx.
//
// Parameters:
//   - ctx: cancellation and deadline for the request.
//
// Returns:
//   - *PLoss: the counts.
//   - error: [ErrEmptyConn] when not connected, [ErrInvalidPacketLossResponse] for a malformed reply, the
//     context's error when it ends the request, or the I/O error.
func (client *Client) PacketLossContext(ctx context.Context) (*PLoss, error) {
	if client.conn == nil {
		return nil, ErrEmptyConn
	}

	defer client.clearDeadline()

	stop := client.watchContext(ctx)
	defer stop()

	err := client.setDeadline(ctx)
	if err != nil {
		return nil, err
	}

	result, err := client.exchange(ctx, packetLoss)
	if err != nil {
		return nil, err
	}

	return parsePLoss(result)
}

// Download asks the server for size bytes and passes them to handler as a stream, observing ctx.
//
// The handler may stop reading early. The rest of the bytes are then read and discarded, so the next command on the
// connection gets its own reply. A transfer that fails after its command is sent closes the connection, since the
// stream is left at an unknown point. The context's deadline applies to the whole transfer, and cancelling the
// context closes the connection.
//
// Parameters:
//   - ctx: cancellation and deadline for the transfer.
//   - size: the number of bytes to request.
//   - handler: reads the downloaded bytes. The reader ends after size bytes.
//
// Returns:
//   - error: an error wrapping [ErrInvalidTransfer] for a non-positive size or nil handler, [ErrEmptyConn] when not
//     connected, the context's error when it ended the transfer, the handler's error, or [io.ErrUnexpectedEOF] when
//     the server sends fewer bytes.
func (client *Client) Download(
	ctx context.Context,
	size int64,
	handler func(io.Reader) error,
) error {
	if size <= 0 || handler == nil {
		return fmt.Errorf("%w: download of %d bytes", ErrInvalidTransfer, size)
	}

	if client.conn == nil {
		return ErrEmptyConn
	}

	defer client.clearDeadline()

	stop := client.watchContext(ctx)
	defer stop()

	err := client.setDeadline(ctx)
	if err != nil {
		return err
	}

	err = client.Write(downloadFormat(size))
	if err != nil {
		return client.abandon(ctx, err)
	}

	stream := &io.LimitedReader{R: client.reader, N: size}

	err = handler(stream)
	if err != nil {
		return client.abandon(ctx, fmt.Errorf("download handler failed: %w", err))
	}

	_, err = io.Copy(io.Discard, stream)
	if err != nil {
		return client.abandon(ctx, fmt.Errorf("failed to read download: %w", err))
	}

	if stream.N > 0 {
		return client.abandon(ctx, fmt.Errorf("download ended after %d of %d bytes: %w",
			size-stream.N, size, io.ErrUnexpectedEOF))
	}

	return nil
}

// Upload sends an upload of a declared total size with its payload read from src, observing ctx.
//
// It sends "UPLOAD <size> 0", then the payload of [UploadPayloadSize] bytes and a final newline, and reads the
// server's "OK <bytes> <timestamp>" confirmation. A transfer that fails before the confirmation is read closes the
// connection, since the stream is left at an unknown point. The context's deadline applies to the whole transfer,
// and cancelling the context closes the connection.
//
// Parameters:
//   - ctx: cancellation and deadline for the transfer.
//   - size: the declared total size, including the command line and the final newline.
//   - src: supplies the payload.
//
// Returns:
//   - int64: the byte count the server confirmed.
//   - error: an error wrapping [ErrInvalidTransfer] for an unusable size or nil source, [ErrEmptyConn] when not
//     connected, the context's error when it ended the transfer, [io.ErrUnexpectedEOF] when src runs out early, or
//     an error wrapping [ErrInvalidResponse] for a malformed confirmation.
func (client *Client) Upload(ctx context.Context, size int64, src io.Reader) (int64, error) {
	if src == nil {
		return 0, fmt.Errorf("%w: upload without a source", ErrInvalidTransfer)
	}

	payload, err := UploadPayloadSize(size)
	if err != nil {
		return 0, err
	}

	if client.conn == nil {
		return 0, ErrEmptyConn
	}

	defer client.clearDeadline()

	stop := client.watchContext(ctx)
	defer stop()

	err = client.setDeadline(ctx)
	if err != nil {
		return 0, err
	}

	err = client.Write(uploadFormat(size))
	if err != nil {
		return 0, client.abandon(ctx, err)
	}

	_, err = io.CopyN(client.conn, src, payload)
	if errors.Is(err, io.EOF) {
		return 0, client.abandon(ctx, fmt.Errorf(
			"upload source ended before %d bytes: %w",
			payload,
			io.ErrUnexpectedEOF,
		))
	}

	if err != nil {
		return 0, client.abandon(ctx, fmt.Errorf("failed to send upload: %w", err))
	}

	err = client.writeAll([]byte{'\n'})
	if err != nil {
		return 0, client.abandon(ctx, err)
	}

	reply, err := client.Read()
	if err != nil {
		return 0, client.abandon(ctx, err)
	}

	// A malformed confirmation was still read as a whole line, so the connection stays usable.
	return parseUploadReply(reply, size)
}

// abandon closes and forgets the connection after a transfer failed partway, and reports the failure.
//
// Parameters:
//   - ctx: the transfer's context.
//   - err: the failure.
//
// Returns:
//   - error: the context's error when it ended the transfer, and err otherwise.
func (client *Client) abandon(ctx context.Context, err error) error {
	if client.conn != nil {
		_ = client.conn.Close()
	}

	client.conn = nil
	client.reader = nil
	client.version = ""

	return contextError(ctx, err)
}

// exchange sends one command and reads its reply, reporting the context's error when it ended the exchange.
//
// Parameters:
//   - ctx: the operation's context.
//   - command: the command without its newline.
//
// Returns:
//   - []byte: the reply line, including its newline.
//   - error: the context's error or the I/O error.
func (client *Client) exchange(ctx context.Context, command []byte) ([]byte, error) {
	err := client.Write(command)
	if err != nil {
		return nil, contextError(ctx, err)
	}

	reply, err := client.Read()
	if err != nil {
		return nil, contextError(ctx, err)
	}

	return reply, nil
}

// writeAll writes data in full, looping over short writes.
//
// Parameters:
//   - data: the bytes to write.
//
// Returns:
//   - error: the write error, or [io.ErrShortWrite] when a write makes no progress.
func (client *Client) writeAll(data []byte) error {
	for len(data) > 0 {
		written, err := client.conn.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write to connection: %w", err)
		}

		if written == 0 {
			return io.ErrShortWrite
		}

		data = data[written:]
	}

	return nil
}

// setDeadline applies the context's deadline to the connection, or clears any earlier deadline.
//
// Parameters:
//   - ctx: the context whose deadline to apply.
//
// Returns:
//   - error: [ErrEmptyConn] when not connected, or the error from setting the deadline.
func (client *Client) setDeadline(ctx context.Context) error {
	if client.conn == nil {
		return ErrEmptyConn
	}

	deadline, _ := ctx.Deadline()

	err := client.conn.SetDeadline(deadline)
	if err != nil {
		return fmt.Errorf("failed to set connection deadline: %w", err)
	}

	return nil
}

// clearDeadline removes any deadline an operation set, so later calls that set none are not cut off by it.
func (client *Client) clearDeadline() {
	if client.conn != nil {
		_ = client.conn.SetDeadline(time.Time{})
	}
}

// watchContext closes the connection when ctx ends before the operation finishes, which unblocks any read or write
// in progress.
//
// Completion and cancellation race to claim the outcome. When stop claims it first, the watcher never closes the
// connection, so a finished operation is never cut off late. When the watcher claims it first, it closes the
// connection, and stop drops that connection from the client. stop waits for the watcher to exit, and clears the
// client's fields on the caller's goroutine, so nothing races with the operation.
//
// Parameters:
//   - ctx: the context to watch.
//
// Returns:
//   - func(): ends the watch. Call it when the operation finishes, and use the client only after it returns.
func (client *Client) watchContext(ctx context.Context) func() {
	if ctx.Done() == nil {
		return func() {}
	}

	const (
		running = iota
		completed
		cancelled
	)

	var state atomic.Int32

	conn := client.conn
	done := make(chan struct{})
	exited := make(chan struct{})

	go func() {
		defer close(exited)

		select {
		case <-ctx.Done():
			if state.CompareAndSwap(running, cancelled) {
				_ = conn.Close()
			}
		case <-done:
		}
	}()

	return func() {
		state.CompareAndSwap(running, completed)
		close(done)
		<-exited

		if state.Load() == cancelled && client.conn == conn {
			client.conn = nil
			client.reader = nil
			client.version = ""
		}
	}
}

// contextError reports the context's error when the context ended the operation, and err otherwise.
//
// The connection deadline equals the context deadline, so the socket can time out a moment before the context
// reports that its deadline passed. A socket timeout at or after the context deadline is therefore reported as
// [context.DeadlineExceeded] too.
//
// Parameters:
//   - ctx: the operation's context.
//   - err: the I/O error the operation returned.
//
// Returns:
//   - error: a wrapped context error when ctx is done or its deadline has passed, or err.
func contextError(ctx context.Context, err error) error {
	ctxErr := ctx.Err()
	if ctxErr != nil {
		return fmt.Errorf("operation ended by context: %w", ctxErr)
	}

	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline && errors.Is(err, os.ErrDeadlineExceeded) && !time.Now().Before(deadline) {
		return fmt.Errorf("operation ended by context: %w: %w", context.DeadlineExceeded, err)
	}

	return err
}

// parseHello extracts the server version from a HELLO reply.
//
// Parameters:
//   - message: the reply line, including its newline.
//
// Returns:
//   - string: the version text after "HELLO ".
//   - error: [ErrInvalidResponse] when the reply is not HELLO or carries no version.
func parseHello(message []byte) (string, error) {
	version, found := bytes.CutPrefix(bytes.TrimRight(message, "\r\n"), helloPrefix)
	if !found || len(version) == 0 {
		return "", fmt.Errorf("%w: %q", ErrInvalidResponse, message)
	}

	return string(version), nil
}

// parsePong extracts the server's clock from a "PONG <milliseconds>" reply.
//
// Parameters:
//   - reply: the reply line, ending in LF or CRLF.
//
// Returns:
//   - int64: the server's clock in milliseconds.
//   - error: an error wrapping [ErrEchoData] when the reply is not PONG or its clock is not a number.
func parsePong(reply []byte) (int64, error) {
	stamp, found := bytes.CutPrefix(bytes.TrimRight(reply, "\r\n"), pongPrefix)
	if !found {
		return 0, fmt.Errorf("%w: %q", ErrEchoData, reply)
	}

	milliseconds, err := strconv.ParseInt(string(stamp), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrEchoData, err)
	}

	return milliseconds, nil
}

// parsePLoss parses a "PLOSS <sent> <dup> <max>" reply.
//
// Parameters:
//   - result: the reply line, including its newline.
//
// Returns:
//   - *PLoss: the counts.
//   - error: [ErrInvalidPacketLossResponse] when the reply is not PLOSS, or a parse error for a count.
func parsePLoss(result []byte) (*PLoss, error) {
	splitResult := bytes.Split(bytes.TrimRight(result, "\r\n"), []byte{0x20})
	if len(splitResult) < 4 || !bytes.Equal(splitResult[0], packetLoss) {
		return nil, ErrInvalidPacketLossResponse
	}

	sent, err := strconv.Atoi(string(splitResult[1]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse sent packets: %w", err)
	}

	dup, err := strconv.Atoi(string(splitResult[2]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse duplicate packets: %w", err)
	}

	highest, err := strconv.Atoi(string(splitResult[3]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse max packet index: %w", err)
	}

	return &PLoss{Sent: sent, Dup: dup, Max: highest}, nil
}

// parseUploadReply parses the server's "OK <bytes> <timestamp>" confirmation of an upload.
//
// Parameters:
//   - reply: the reply line, including its newline.
//   - size: the declared total size of the upload.
//
// Returns:
//   - int64: the confirmed byte count.
//   - error: an error wrapping [ErrInvalidResponse] when the reply is not OK or its count is outside 0 to size.
func parseUploadReply(reply []byte, size int64) (int64, error) {
	fields := bytes.Fields(reply)
	if len(fields) != 3 || !bytes.Equal(fields[0], okReply) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidResponse, reply)
	}

	confirmed, err := strconv.ParseInt(string(fields[1]), 10, 64)
	if err != nil || confirmed < 0 || confirmed > size {
		return 0, fmt.Errorf("%w: %q", ErrInvalidResponse, reply)
	}

	return confirmed, nil
}

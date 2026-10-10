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

var (
	pingPrefix = []byte{0x50, 0x49, 0x4e, 0x47, 0x20}
	// downloadPrefix = []byte{0x44, 0x4F, 0x57, 0x4E, 0x4C, 0x4F, 0x41, 0x44, 0x20}
	// uploadPrefix   = []byte{0x55, 0x50, 0x4C, 0x4F, 0x41, 0x44, 0x20}.
	initPacket = []byte{0x49, 0x4e, 0x49, 0x54, 0x50, 0x4c, 0x4f, 0x53, 0x53}
	packetLoss = []byte{0x50, 0x4c, 0x4f, 0x53, 0x53}
	hiFormat   = []byte{0x48, 0x49}
	quitFormat = []byte{0x51, 0x55, 0x49, 0x54}
)

var (
	// ErrEchoData is returned when the echo data is incorrect.
	ErrEchoData = errors.New("incorrect echo data")
	// ErrEmptyConn is returned when the connection is empty.
	ErrEmptyConn = errors.New("empty conn")
	// ErrUnsupported is returned when the protocol is unsupported.
	ErrUnsupported = errors.New(
		"unsupported protocol",
	) // Some servers have disabled ip:8080, we return this error.
	// ErrUninitializedPacketLossInst is returned when the packet loss instance is not initialized.
	ErrUninitializedPacketLossInst = errors.New("uninitialized packet loss inst")
	// ErrInvalidPacketLossResponse is returned when the packet loss response is invalid.
	ErrInvalidPacketLossResponse = errors.New("invalid packet loss response")
	// ErrInvalidResponse is returned when the server's reply does not follow the protocol.
	ErrInvalidResponse = errors.New("invalid TCP speedtest response")
	// ErrNilDialer is returned when a client is created without a dialer and asked to connect.
	ErrNilDialer = errors.New("transport client has no dialer")
)

// Server replies checked during the handshake and packet-loss setup.
var (
	// helloPrefix starts the server's reply to HI and to "HI <uuid>".
	helloPrefix = []byte("HELLO ")
	// okReply is the server's reply to INITPLOSS.
	okReply = []byte("OK")
)

// pingFormat builds a PING command for a local timestamp, copying the shared prefix so callers never alias it.
func pingFormat(locTime int64) []byte {
	return strconv.AppendInt(bytes.Clone(pingPrefix), locTime, 10)
}

// Client represents a TCP client for speedtest operations.
type Client struct {
	id      string
	conn    net.Conn
	host    string
	version string

	dialer *net.Dialer

	reader *bufio.Reader
}

// NewClient creates a new TCP client with the given dialer.
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

		if len(data) != 19 {
			return 0, ErrEchoData
		}

		tx, err := strconv.ParseInt(string(data[5:18]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", ErrEchoData, err)
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

// PLoss Packet loss statistics
// The packet loss here generally refers to uplink packet loss.
// We use the following formula to calculate the packet loss:
// packetLoss = [1 - (Sent - Dup) / (Max + 1)] * 100%.
type PLoss struct {
	Sent int `json:"sent"` // Number of sent packets acknowledged by the remote.
	Dup  int `json:"dup"`  // Number of duplicate packets acknowledged by the remote.
	Max  int `json:"max"`  // The maximum index value received by the remote.
}

func (p PLoss) String() string {
	if p.Sent == 0 {
		// if p.Sent == 0, maybe all data is dropped by the upper gateway.
		// we believe this feature is not applicable on this server now.
		return "Packet Loss: N/A"
	}

	return fmt.Sprintf(
		"Packet Loss: %.2f%% (Sent: %d/Dup: %d/Max: %d)",
		p.Loss()*100,
		p.Sent,
		p.Dup,
		p.Max,
	)
}

// Loss returns the packet loss ratio.
func (p PLoss) Loss() float64 {
	if p.Sent == 0 {
		return -1
	}

	return 1 - float64(p.Sent-p.Dup)/float64(p.Max+1)
}

// LossPercent returns the packet loss percentage.
func (p PLoss) LossPercent() float64 {
	if p.Sent == 0 {
		return -1
	}

	return p.Loss() * 100
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

// Download performs a download test. Currently unimplemented.
func (client *Client) Download() {
	panic("Unimplemented method: Client.Download()")
}

// Upload performs an upload test. Currently unimplemented.
func (client *Client) Upload() {
	panic("Unimplemented method: Client.Upload()")
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

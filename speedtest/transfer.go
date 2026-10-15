// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"fmt"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/meter"
)

// Transfer is the result of a download or upload test.
type Transfer struct {
	// Connections are the measurements of each connection the test used, in the order they started.
	Connections []ConnectionStats
	// Rate is the bytes per second over the test's capture window, or [StrNotAvailable]'s -1 when no data was
	// counted and more than a tenth of the requests failed.
	Rate ByteRate
	// Bytes is the bytes downloaded, or the upload bytes the servers confirmed, when the test stopped. It is the sum
	// of the connections' bytes.
	Bytes int64
	// Sent is the upload bytes read into the transport when the test stopped, confirmed or not. It is 0 for
	// downloads.
	Sent int64
	// Window is the capture window the rate is measured over.
	Window time.Duration
	// Duration is how long the test took, including the requests that were cut short when it stopped.
	Duration time.Duration
	// Requests is the number of requests the test sent.
	Requests int
	// Failures is the number of requests that failed.
	Failures int
}

// ConnectionStats is the measurement of one connection of a download or upload test.
type ConnectionStats struct {
	// ServerID is the ID of the server the connection sent its requests to.
	ServerID string
	// Bytes is the bytes the connection downloaded, or the upload bytes the server confirmed, when the test stopped.
	Bytes int64
	// Requests is the number of requests the connection sent.
	Requests int
	// Failures is the number of the connection's requests that failed.
	Failures int
	// Active is the time the connection spent in requests before the test stopped. Upload connections can wait
	// idle while fewer of them are needed.
	Active time.Duration
}

// TransferOption configures one download or upload test.
type TransferOption func(*transferSettings)

// transferSettings is the configuration of one download or upload test, collected from its options.
type transferSettings struct {
	// onRate receives the live rate while the test runs, or is nil.
	onRate func(rate ByteRate)
}

// transferKind is how a download or an upload test registers and sends its requests.
type transferKind struct {
	// register adds a request handler to the test's direction.
	register func(transferMeter *meter.Meter, handler meter.Handler) *meter.Direction
	// request sends one request.
	request requestFunc
	// name names the direction in debug messages.
	name string
	// resolveUploads resolves each server's upload redirects before the test starts.
	resolveUploads bool
}

// failureShareForNotAvailable is the share of failed requests above which a test that counted no data reports its
// rate as not available rather than zero.
const failureShareForNotAvailable = 0.1

// WithRateCallback sets a function that receives the live rate estimate while a download or upload test runs.
//
// Parameters:
//   - callback: the function, or nil for none.
//
// Returns:
//   - TransferOption: the option.
func WithRateCallback(callback func(rate ByteRate)) TransferOption {
	return func(cfg *transferSettings) {
		cfg.onRate = callback
	}
}

// ConfirmationRatio returns the share of the upload bytes read into the transport that the servers confirmed. It
// counts application-level confirmations, not TCP acknowledgments.
//
// Returns:
//   - float64: the share from 0 to 1, or 0 when nothing was sent, as for downloads.
func (t Transfer) ConfirmationRatio() float64 {
	if t.Sent <= 0 {
		return 0
	}

	return min(max(float64(t.Bytes)/float64(t.Sent), 0), 1)
}

// Rate returns the connection's throughput while it was sending requests.
//
// Returns:
//   - ByteRate: the bytes per second over its active time, or 0 when it was never active.
func (c ConnectionStats) Rate() ByteRate {
	if c.Active <= 0 {
		return 0
	}

	return ByteRate(float64(c.Bytes) / c.Active.Seconds())
}

// Download measures the download speed from a server.
//
// Parameters:
//   - ctx: ends the test early, which makes it fail.
//   - server: the server.
//   - opts: options for this test.
//
// Returns:
//   - Transfer: the measurement.
//   - error: [ErrServerNil], [ErrUninitializedManager] for a client not built by [New], or the context's error.
func (s *Client) Download(
	ctx context.Context,
	server *Server,
	opts ...TransferOption,
) (Transfer, error) {
	if !s.ready() {
		return Transfer{}, ErrUninitializedManager
	}

	if server == nil {
		return Transfer{}, ErrServerNil
	}

	return s.transfer(
		ctx,
		s.downloadKind(),
		Servers{server},
		s.downloadIndex(defaultDownloadIndex),
		opts,
	)
}

// Upload measures the upload speed to a server.
//
// Parameters:
//   - ctx: ends the test early, which makes it fail.
//   - server: the server.
//   - opts: options for this test.
//
// Returns:
//   - Transfer: the measurement.
//   - error: [ErrServerNil], [ErrUninitializedManager] for a client not built by [New], or the context's error.
func (s *Client) Upload(
	ctx context.Context,
	server *Server,
	opts ...TransferOption,
) (Transfer, error) {
	if !s.ready() {
		return Transfer{}, ErrUninitializedManager
	}

	if server == nil {
		return Transfer{}, ErrServerNil
	}

	return s.transfer(
		ctx,
		s.uploadKind(),
		Servers{server},
		s.uploadIndex(defaultUploadIndex),
		opts,
	)
}

// DownloadMulti measures the download speed from several servers at once. The main server gets the main share of
// the connections, and the other available servers in the pool share the rest.
//
// Parameters:
//   - ctx: ends the test early, which makes it fail.
//   - main: the main server.
//   - pool: the servers to share the test with, of which those with a measured latency are used.
//   - opts: options for this test.
//
// Returns:
//   - Transfer: the measurement, with each connection naming its server.
//   - error: [ErrServerNil], [ErrNoAvailableServers], [ErrUninitializedManager] for a client not built by [New], or
//     the context's error.
func (s *Client) DownloadMulti(
	ctx context.Context,
	main *Server,
	pool Servers,
	opts ...TransferOption,
) (Transfer, error) {
	if !s.ready() {
		return Transfer{}, ErrUninitializedManager
	}

	servers, err := multiServers(main, pool)
	if err != nil {
		return Transfer{}, err
	}

	return s.transfer(
		ctx,
		s.downloadKind(),
		servers,
		s.downloadIndex(defaultDownloadIndex),
		opts,
	)
}

// UploadMulti measures the upload speed to several servers at once, sharing the connections like
// [Client.DownloadMulti].
//
// Parameters:
//   - ctx: ends the test early, which makes it fail.
//   - main: the main server.
//   - pool: the servers to share the test with, of which those with a measured latency are used.
//   - opts: options for this test.
//
// Returns:
//   - Transfer: the measurement, with each connection naming its server.
//   - error: [ErrServerNil], [ErrNoAvailableServers], [ErrUninitializedManager] for a client not built by [New], or
//     the context's error.
func (s *Client) UploadMulti(
	ctx context.Context,
	main *Server,
	pool Servers,
	opts ...TransferOption,
) (Transfer, error) {
	if !s.ready() {
		return Transfer{}, ErrUninitializedManager
	}

	servers, err := multiServers(main, pool)
	if err != nil {
		return Transfer{}, err
	}

	return s.transfer(
		ctx,
		s.uploadKind(),
		servers,
		s.uploadIndex(defaultMultiUploadIndex),
		opts,
	)
}

// transfer runs a download or upload test on its own meter, so no counts carry over between tests.
//
// Parameters:
//   - ctx: ends the test early, which makes it fail.
//   - kind: how the test registers and sends its requests.
//   - servers: the servers, each of which gets a request handler, with the main server first.
//   - size: the index into the size table.
//   - opts: options for this test.
//
// Returns:
//   - Transfer: the measurement.
//   - error: [ErrUninitializedManager] for a client not built by [New], or the context's error.
func (s *Client) transfer(
	ctx context.Context,
	kind transferKind,
	servers Servers,
	size int,
	opts []TransferOption,
) (Transfer, error) {
	if !s.ready() {
		return Transfer{}, ErrUninitializedManager
	}

	cfg := transferSettings{onRate: nil}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Resolve before the handlers are registered, so no upload worker sees a URL change. TCP tests use no HTTP
	// upload URL.
	uploadURLs := make([]string, len(servers))
	if kind.resolveUploads && s.settings.testMode != TCPTest {
		uploadURLs = s.resolveUploadURLs(ctx, servers, maxConcurrentUploadResolves)
	}

	transferMeter := meter.New(meter.Config{
		Logger:               s.settings.logger,
		OnRate:               rateCallback(cfg.onRate),
		Connections:          s.settings.connections,
		CaptureTime:          s.settings.testDuration,
		RateCaptureFrequency: 0,
	})

	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var direction *meter.Direction

	for index, server := range servers {
		request := &transferRequest{
			client:    s,
			server:    server,
			uploadURL: uploadURLs[index],
			size:      size,
		}

		s.log().
			Debug(ctx, "registering transfer handler", logKeyDirection, kind.name, logKeyURL, server.URL)

		direction = kind.register(transferMeter, func(connection *meter.Connection) error {
			return kind.request(requestCtx, request, connection)
		})
	}

	start := time.Now()

	direction.Run(ctx, cancel, 0)

	duration := time.Since(start)

	// A cancelled test measured nothing useful.
	err := ctx.Err()
	if err != nil {
		return Transfer{}, fmt.Errorf("speed test ended by context: %w", err)
	}

	return newTransfer(direction.Result(), servers, duration), nil
}

// downloadKind returns how download tests register and send their requests in the client's test mode.
//
// Returns:
//   - transferKind: TCP requests in [TCPTest] mode, and HTTP requests otherwise.
func (s *Client) downloadKind() transferKind {
	request := requestFunc(downloadRequest)
	if s.settings.testMode == TCPTest {
		request = tcpDownloadRequest
	}

	return transferKind{
		register:       (*meter.Meter).RegisterDownload,
		request:        request,
		name:           "Download",
		resolveUploads: false,
	}
}

// uploadKind returns how upload tests register and send their requests in the client's test mode.
//
// Returns:
//   - transferKind: TCP requests in [TCPTest] mode, and HTTP requests otherwise.
func (s *Client) uploadKind() transferKind {
	request := requestFunc(uploadRequest)
	if s.settings.testMode == TCPTest {
		request = tcpUploadRequest
	}

	return transferKind{
		register:       (*meter.Meter).RegisterUpload,
		request:        request,
		name:           "Upload",
		resolveUploads: true,
	}
}

// ready reports whether the client was built by [New], so it can send requests.
//
// Returns:
//   - bool: true for a client built by New.
func (s *Client) ready() bool {
	return s != nil && s.doer != nil
}

// multiServers returns the servers a multi-server test uses, with its main server first.
//
// Parameters:
//   - main: the main server.
//   - pool: the servers to share the test with.
//
// Returns:
//   - Servers: the pool's servers with a measured latency, fastest first, except that the main server comes first
//     when it is among them. A test registers handlers only up to its connection count, so the main server must not
//     be left out.
//   - error: [ErrServerNil] for a nil main server, or [ErrNoAvailableServers] when no server has a latency.
func multiServers(main *Server, pool Servers) (Servers, error) {
	if main == nil {
		return nil, ErrServerNil
	}

	available := *pool.Available()
	if len(available) == 0 {
		return nil, ErrNoAvailableServers
	}

	for index, server := range available {
		if server.ID == main.ID {
			ordered := append(Servers{server}, available[:index]...)

			return append(ordered, available[index+1:]...), nil
		}
	}

	return available, nil
}

// newTransfer builds a transfer result from the meter's measurement.
//
// Parameters:
//   - result: the meter's measurement.
//   - servers: the servers, in handler order.
//   - duration: how long the test took.
//
// Returns:
//   - Transfer: the result.
func newTransfer(result meter.Result, servers Servers, duration time.Duration) Transfer {
	rate := ByteRate(result.Rate)
	if rate == 0 && result.Requests > 0 &&
		float64(result.Failures)/float64(result.Requests) > failureShareForNotAvailable {
		rate = -1
	}

	connections := make([]ConnectionStats, 0, len(result.Connections))
	for _, connection := range result.Connections {
		connections = append(connections, ConnectionStats{
			ServerID: servers[connection.Handler].ID,
			Bytes:    connection.Bytes,
			Requests: connection.Requests,
			Failures: connection.Failures,
			Active:   connection.Active,
		})
	}

	return Transfer{
		Connections: connections,
		Rate:        rate,
		Bytes:       result.Bytes,
		Sent:        result.Sent,
		Window:      result.Window,
		Duration:    duration,
		Requests:    result.Requests,
		Failures:    result.Failures,
	}
}

// rateCallback adapts a rate callback to the meter's, which reports bytes per second.
//
// Parameters:
//   - callback: the callback, or nil.
//
// Returns:
//   - func(rate float64): the adapted callback, or nil for a nil callback.
func rateCallback(callback func(rate ByteRate)) func(rate float64) {
	if callback == nil {
		return nil
	}

	return func(rate float64) { callback(ByteRate(rate)) }
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/transport"
)

// PingStats is the latency measured by [Client.Ping].
type PingStats struct {
	// Latencies are the measured round trips, in the order they were sent.
	Latencies []time.Duration
	// Latency is the mean round trip.
	Latency time.Duration
	// Jitter is the standard deviation of the round trips.
	Jitter time.Duration
	// Min is the shortest round trip.
	Min time.Duration
	// Max is the longest round trip.
	Max time.Duration
	// Duration is how long the measurement took.
	Duration time.Duration
}

// PingOption configures one call of [Client.Ping].
type PingOption func(*pingSettings)

// pingSettings is the configuration of one call of [Client.Ping], collected from its options.
type pingSettings struct {
	// onLatency receives each round trip as it is measured, or is nil.
	onLatency func(latency time.Duration)
	// count is the number of round trips to measure.
	count int
	// interval is the wait after each round trip.
	interval time.Duration
}

// Ping defaults.
const (
	// defaultPingCount is the number of round trips a ping measures.
	defaultPingCount = 10
	// defaultPingInterval is the wait after each round trip.
	defaultPingInterval = 200 * time.Millisecond
	// icmpReadTimeout bounds the wait for each ICMP echo reply.
	icmpReadTimeout = 4 * time.Second
)

// ICMP and probe constants.
const (
	// PingTimeout is the [Server.Latency] of a server whose latency probe failed.
	PingTimeout = -1
	// echoOptionDataSize is the size of an echo request's data, which holds the echo message.
	echoOptionDataSize = 32
	// icmpEchoReply is the ICMP message type of an echo reply.
	icmpEchoReply = 0
	// maxIPv4HeaderSize is the largest IPv4 header, which precedes the ICMP message on a raw socket.
	maxIPv4HeaderSize = 60
)

// WithPingCount sets how many round trips [Client.Ping] measures. The default is 10.
//
// Parameters:
//   - count: the number of round trips, at least 1.
//
// Returns:
//   - PingOption: the option.
func WithPingCount(count int) PingOption {
	return func(cfg *pingSettings) {
		cfg.count = count
	}
}

// WithPingInterval sets how long [Client.Ping] waits after each round trip. The default is 200 milliseconds.
//
// Parameters:
//   - interval: the wait, 0 or more.
//
// Returns:
//   - PingOption: the option.
func WithPingInterval(interval time.Duration) PingOption {
	return func(cfg *pingSettings) {
		cfg.interval = interval
	}
}

// WithLatencyCallback sets a function that receives each round trip as [Client.Ping] measures it.
//
// Parameters:
//   - callback: the function, or nil for none.
//
// Returns:
//   - PingOption: the option.
func WithLatencyCallback(callback func(latency time.Duration)) PingOption {
	return func(cfg *pingSettings) {
		cfg.onLatency = callback
	}
}

// Ping measures the latency to a server with the client's ping mode: HTTP requests by default, or the
// speedtest.net TCP protocol or ICMP echo requests with [WithPingMode].
//
// Parameters:
//   - ctx: ends the measurement early, which makes it fail.
//   - server: the server.
//   - opts: options for this call.
//
// Returns:
//   - PingStats: the latency.
//   - error: [ErrServerNil], [ErrUninitializedManager] for a client not built by [New], an error wrapping
//     [ErrInvalidOption] for an invalid option, [ErrConnectTimeout] when no round trip succeeded, or the context's
//     error when ctx ended first.
func (s *Client) Ping(ctx context.Context, server *Server, opts ...PingOption) (PingStats, error) {
	if !s.ready() {
		return PingStats{}, ErrUninitializedManager
	}

	if server == nil {
		return PingStats{}, ErrServerNil
	}

	cfg := pingSettings{onLatency: nil, count: defaultPingCount, interval: defaultPingInterval}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.count < 1 || cfg.interval < 0 {
		return PingStats{}, fmt.Errorf(
			"%w: a ping needs a count of at least 1 and an interval of 0 or more, not %d and %s",
			ErrInvalidOption,
			cfg.count,
			cfg.interval,
		)
	}

	start := time.Now()

	latencies, err := s.pingWithMode(ctx, server, cfg)
	if err != nil {
		return PingStats{}, err
	}

	// A ping cut short by its context measured too little to report.
	err = ctx.Err()
	if err != nil {
		return PingStats{}, fmt.Errorf("ping ended by context: %w", err)
	}

	if len(latencies) == 0 {
		return PingStats{}, ErrConnectTimeout
	}

	s.log().Debug(ctx, "measured latencies", logKeyLatencies, latencies)

	return pingStats(latencies, time.Since(start)), nil
}

// pingWithMode measures round trips with the client's ping mode.
//
// Parameters:
//   - ctx: ends the measurement early.
//   - server: the server.
//   - cfg: the call's settings.
//
// Returns:
//   - []int64: the round trips in nanoseconds.
//   - error: the ping's error.
func (s *Client) pingWithMode(
	ctx context.Context,
	server *Server,
	cfg pingSettings,
) ([]int64, error) {
	switch s.settings.pingMode {
	case TCP:
		return s.tcpPing(ctx, server, cfg.count, cfg.interval, cfg.onLatency)
	case ICMP:
		return s.icmpPing(ctx, server, icmpReadTimeout, cfg.count, cfg.interval, cfg.onLatency)
	case HTTP:
		return s.httpPing(ctx, server, cfg.count, cfg.interval, cfg.onLatency)
	default:
		return s.httpPing(ctx, server, cfg.count, cfg.interval, cfg.onLatency)
	}
}

// pingStats summarizes measured round trips.
//
// Parameters:
//   - latencies: the round trips in nanoseconds, at least one.
//   - duration: how long the measurement took.
//
// Returns:
//   - PingStats: the summary.
func pingStats(latencies []int64, duration time.Duration) PingStats {
	mean, _, jitter, minLatency, maxLatency := StandardDeviation(latencies)

	measured := make([]time.Duration, 0, len(latencies))
	for _, latency := range latencies {
		measured = append(measured, time.Duration(latency))
	}

	return PingStats{
		Latencies: measured,
		Latency:   time.Duration(mean),
		Jitter:    time.Duration(jitter),
		Min:       time.Duration(minLatency),
		Max:       time.Duration(maxLatency),
		Duration:  duration,
	}
}

// tcpPing measures latency with the speedtest.net TCP protocol's PING command.
//
// Parameters:
//   - ctx: ends the pings early.
//   - server: the server.
//   - echoTimes: the number of pings.
//   - echoFreq: the wait after each ping.
//   - callback: receives each latency, or is nil.
//
// Returns:
//   - []int64: the latencies in nanoseconds.
//   - error: a connection error, [ErrConnectTimeout] when every ping failed, or the context's error.
func (s *Client) tcpPing(
	ctx context.Context,
	server *Server,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrUninitializedManager
	}

	if server == nil {
		return nil, ErrServerNil
	}

	var pingDst string

	if len(server.Host) == 0 {
		u, err := url.Parse(server.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server URL for TCP ping: %w", err)
		}

		if len(u.Host) == 0 {
			return nil, fmt.Errorf("invalid server URL %q for TCP ping: %w", server.URL, errNoHost)
		}

		pingDst = u.Host
	} else {
		pingDst = server.Host
	}

	failTimes := 0

	latencies := make([]int64, 0, echoTimes)

	client, err := transport.NewClient(s.tcpDialer)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport client for TCP ping: %w", err)
	}

	err = client.Connect(ctx, pingDst)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for TCP ping: %w", err)
	}

	defer func() { _ = client.Disconnect() }()

	for range echoTimes {
		latency, err := client.PingContext(ctx)
		if err != nil {
			if isContextError(err) {
				return latencies, fmt.Errorf("TCP ping ended by context: %w", err)
			}

			failTimes++

			continue
		}

		latencies = append(latencies, latency)
		if callback != nil {
			callback(time.Duration(latency))
		}

		err = sleepContext(ctx, echoFreq)
		if err != nil {
			return latencies, err
		}
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	return latencies, nil
}

// httpPing measures latency with HTTP requests for the server's latency.txt. The first request warms up the
// connection and is not counted.
//
// Parameters:
//   - ctx: ends the pings early.
//   - server: the server.
//   - echoTimes: the number of counted pings.
//   - echoFreq: the wait after each ping.
//   - callback: receives each latency, or is nil.
//
// Returns:
//   - []int64: the latencies in nanoseconds.
//   - error: an error wrapping [ErrConnectTimeout] when every request failed, or the context's error.
func (s *Client) httpPing(
	ctx context.Context,
	server *Server,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrUninitializedManager
	}

	if server == nil {
		return nil, ErrServerNil
	}

	var contextErr, lastErr error

	u, err := url.Parse(server.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL for HTTP ping: %w", err)
	}

	if len(u.Host) == 0 {
		return nil, fmt.Errorf("invalid server URL %q for HTTP ping: %w", server.URL, errNoHost)
	}

	u.Path = path.Dir(u.Path)
	pingDst := u.JoinPath("latency.txt").String()
	s.log().Debug(ctx, "pinging over HTTP", logKeyURL, pingDst)

	failTimes := 0
	latencies := make([]int64, 0, echoTimes+1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingDst, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping HTTP request: %w", err)
	}
	// carry out an extra request to warm up the connection and ensure the first request is not going to affect the
	// overall estimation
	echoTimes++
	for i := range echoTimes {
		sTime := time.Now()
		resp, err := s.doer.Do(req)
		endTime := time.Since(sTime)

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				contextErr = err

				break
			}

			lastErr = err
			failTimes++

			continue
		}

		// A server that answers with an error status must not look reachable, or it could win server selection.
		statusErr := checkStatus(resp)

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if statusErr != nil {
			lastErr = statusErr
			failTimes++

			continue
		}

		if i > 0 {
			latency := endTime.Nanoseconds()
			latencies = append(latencies, latency)
			s.log().Debug(ctx, "measured latency", logKeyLatency, time.Duration(latency))

			if callback != nil {
				callback(endTime)
			}
		}

		err = sleepContext(ctx, echoFreq)
		if err != nil {
			contextErr = err

			break
		}
	}

	if contextErr != nil {
		return latencies, contextErr
	}

	if failTimes == echoTimes {
		return nil, fmt.Errorf("%w: %w", ErrConnectTimeout, lastErr)
	}

	return latencies, nil
}

// icmpPing measures latency with ICMP echo requests, which need permission to open raw sockets.
//
// Parameters:
//   - ctx: ends the pings early.
//   - server: the server.
//   - readTimeout: how long to wait for each reply.
//   - echoTimes: the number of pings.
//   - echoFreq: the wait after each ping.
//   - callback: receives each latency, or is nil.
//
// Returns:
//   - []int64: the latencies in nanoseconds.
//   - error: a dial error, [ErrConnectTimeout] when every ping failed, or the context's error.
func (s *Client) icmpPing(
	ctx context.Context,
	server *Server,
	readTimeout time.Duration,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {
	if s == nil {
		return nil, ErrUninitializedManager
	}

	if server == nil {
		return nil, ErrServerNil
	}

	latencies := make([]int64, 0, echoTimes)

	u, err := url.ParseRequestURI(server.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ICMP URL: %w", err)
	}

	if len(u.Host) == 0 {
		return nil, fmt.Errorf("invalid server URL %q for ICMP ping: %w", server.URL, errNoHost)
	}

	s.log().Debug(ctx, "pinging over ICMP", logKeyAddress, u.Hostname())

	dialContext, err := s.ipDialer.DialContext(
		ctx,
		"ip:icmp",
		strings.Split(u.Host, ":")[0],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ICMP: %w", err)
	}

	defer func() { _ = dialContext.Close() }()

	// Each echo's read has its own deadline, so close the socket when ctx ends to unblock a pending read. Closing
	// cannot be undone by the next echo's deadline, unlike moving the deadline.
	stopClosing := context.AfterFunc(ctx, func() { _ = dialContext.Close() })
	defer stopClosing()

	icmpData := prepareICMPPacket()

	failTimes := 0

	for i := range echoTimes {
		err := ctx.Err()
		if err != nil {
			return latencies, fmt.Errorf("ICMP ping ended by context: %w", err)
		}

		latency, err := sendOneICMPPing(dialContext, icmpData, i, readTimeout)
		if err != nil {
			// A read cut off by the closed socket is the context ending, not a lost echo.
			ctxErr := ctx.Err()
			if ctxErr != nil {
				return latencies, fmt.Errorf("ICMP ping ended by context: %w", ctxErr)
			}

			failTimes++

			continue
		}

		latencies = append(latencies, latency.Nanoseconds())
		s.log().Debug(ctx, "measured latency", logKeyLatency, latency)

		if callback != nil {
			callback(latency)
		}

		err = sleepContext(ctx, echoFreq)
		if err != nil {
			return latencies, err
		}
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	return latencies, nil
}

// prepareICMPPacket returns an ICMP echo request with its identifier, sequence number, and message set, and its
// checksum still zero.
//
// Returns:
//   - []byte: the request.
func prepareICMPPacket() []byte {
	icmpData := make([]byte, 8+echoOptionDataSize) // header + data
	icmpData[0] = 8                                // echo
	icmpData[1] = 0                                // code
	icmpData[2] = 0                                // checksum
	icmpData[3] = 0                                // checksum
	icmpData[4] = 0                                // id
	icmpData[5] = 1                                // id
	icmpData[6] = 0                                // seq
	icmpData[7] = 1                                // seq

	echoMessage := "Hi! SpeedTest-Go \\(●'◡'●)/"
	copy(icmpData[8:], echoMessage)

	icmpData[8+echoOptionDataSize-1] = 6

	return icmpData
}

// sendOneICMPPing sends one echo request on a raw ICMP socket and waits for its reply.
//
// Parameters:
//   - dialContext: the raw ICMP socket.
//   - icmpData: the echo request, whose checksum is filled in here.
//   - readTimeout: how long to wait for the reply.
//
// Returns:
//   - time.Duration: the round trip.
//   - error: the write or read error, including a read past the deadline.
func sendOneICMPPing(dialContext interface {
	Write(data []byte) (n int, err error)
	Read(data []byte) (n int, err error)
	SetDeadline(t time.Time) error
	Close() error
}, icmpData []byte, _ int, readTimeout time.Duration,
) (time.Duration, error) {
	// Update checksum and seq
	icmpData[2] = 0
	icmpData[3] = 0
	icmpData[6] = byte(1 >> 8)
	icmpData[7] = byte(1)
	cs := checkSum(icmpData)
	icmpData[2] = byte(cs >> 8)
	icmpData[3] = byte(cs & 0xFF)

	sTime := time.Now()
	_ = dialContext.SetDeadline(sTime.Add(readTimeout))

	_, err := dialContext.Write(icmpData)
	if err != nil {
		return 0, fmt.Errorf("failed to write ICMP packet: %w", err)
	}

	buf := make([]byte, maxIPv4HeaderSize+8+echoOptionDataSize)

	// A raw ICMP socket receives every ICMP packet addressed to the host, including this client's own request when
	// the target is local. Keep reading until the matching reply arrives or the deadline set above expires.
	for {
		n, err := dialContext.Read(buf)
		if err != nil {
			return 0, fmt.Errorf("failed to read ICMP response: %w", err)
		}

		if isEchoReply(buf[:n], icmpData) {
			return time.Since(sTime), nil
		}
	}
}

// isEchoReply reports whether an IPv4 packet read from a raw ICMP socket is the echo reply to request.
//
// Parameters:
//   - packet: the IPv4 packet, starting with its header.
//   - request: the ICMP echo request that was sent.
//
// Returns:
//   - bool: true when the packet is an echo reply with the request's identifier and sequence number.
func isEchoReply(packet, request []byte) bool {
	if len(packet) == 0 {
		return false
	}

	headerLen := int(packet[0]&0x0f) * 4
	if len(packet) < headerLen+8 {
		return false
	}

	icmp := packet[headerLen:]

	return icmp[0] == icmpEchoReply && bytes.Equal(icmp[4:8], request[4:8])
}

// checkSum computes the RFC 1071 Internet checksum of an ICMP message.
//
// The 16-bit words are added in a wider accumulator and the carries are folded back in, giving the one's complement
// sum that receivers verify. An odd final byte is padded with a zero low byte.
//
// Parameters:
//   - data: the message, with its checksum field set to zero.
//
// Returns:
//   - uint16: the checksum to store in the message.
func checkSum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(data); i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}

	return ^uint16(sum)
}

// sleepContext waits for wait, returning early when ctx ends.
//
// Parameters:
//   - ctx: ends the wait early.
//   - wait: how long to wait.
//
// Returns:
//   - error: a wrapped context error when ctx ended first, or nil.
func sleepContext(ctx context.Context, wait time.Duration) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("ping ended by context: %w", err)
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("ping ended by context: %w", ctx.Err())
	case <-timer.C:
		// When both cases are ready, select picks either one, so a context that ended meanwhile still wins.
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("ping ended by context: %w", err)
		}

		return nil
	}
}

// isContextError reports whether err comes from a cancelled or expired context.
//
// Parameters:
//   - err: the error to check.
//
// Returns:
//   - bool: true when err wraps [context.Canceled] or [context.DeadlineExceeded].
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// StandardDeviation calculates the mean, variance, standard deviation, min, and max of a vector.
//
// Parameters:
//   - vector: the values, such as latencies in nanoseconds.
//
// Returns:
//   - int64: the mean.
//   - int64: the population variance.
//   - int64: the standard deviation.
//   - int64: the smallest value.
//   - int64: the largest value, all 0 for an empty vector.
func StandardDeviation(vector []int64) (int64, int64, int64, int64, int64) {
	if len(vector) == 0 {
		return 0, 0, 0, 0, 0
	}

	var (
		sumNum, mean, minVal, maxVal int64
		accumulate                   float64
	)

	minVal = math.MaxInt64
	maxVal = math.MinInt64

	for _, value := range vector {
		sumNum += value
		if minVal > value {
			minVal = value
		}

		if maxVal < value {
			maxVal = value
		}
	}

	mean = sumNum / int64(len(vector))

	// Squared deviations of nanosecond latencies overflow int64 once values differ by about three seconds, so they
	// are added up as floats.
	for _, value := range vector {
		deviation := float64(value - mean)
		accumulate += deviation * deviation
	}

	variance := accumulate / float64(len(vector))

	// float64(math.MaxInt64) rounds up to 2^63, which converts back to the smallest int64, so a variance that large is
	// clamped explicitly.
	clamped := int64(math.MaxInt64)
	if variance < float64(math.MaxInt64) {
		clamped = int64(variance)
	}

	return mean, clamped, int64(math.Sqrt(variance)), minVal, maxVal
}

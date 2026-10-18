// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/transport"
)

// PacketLossAnalyzerOptions configures the packet loss analyzer.
type PacketLossAnalyzerOptions struct {
	TCPDialer              *net.Dialer // tcp dialer for sampling
	UDPDialer              *net.Dialer // udp dialer for sending packet
	SourceInterface        string      // source interface
	RemoteSamplingInterval time.Duration
	SamplingDuration       time.Duration
	PacketSendingInterval  time.Duration
	PacketSendingTimeout   time.Duration
}

// PacketLossAnalyzer performs packet loss analysis on network connections.
type PacketLossAnalyzer struct {
	options *PacketLossAnalyzerOptions
	// sourceErr is why SourceInterface did not resolve, returned by every run, or nil.
	sourceErr error
}

// PacketLossStats holds the counts a server reports for the datagrams sent to it, which measure uplink packet loss.
//
// The loss is 1 - (Sent - Dup) / (Max + 1), since Max + 1 datagrams were sent by the time the server received the
// one numbered Max. Counts combined from several servers keep that form: their Max is one less than the datagrams
// sent to all of them.
type PacketLossStats struct {
	// Sent is the number of datagrams the server received.
	Sent int `json:"sent"`
	// Dup is the number of duplicate datagrams the server received.
	Dup int `json:"dup"`
	// Max is the highest datagram index the server received.
	Max int `json:"max"`
}

// defaultPacketSendingTimeout bounds connecting the analyzer's sockets when the options set no timeout.
const defaultPacketSendingTimeout = 5 * time.Second

// ErrUnsupported is returned when a server does not serve the packet loss protocol. Some servers close the TCP port
// it uses.
var ErrUnsupported = errors.New("unsupported protocol")

// NewPacketLossAnalyzer creates a new packet loss analyzer with the given options.
//
// SourceInterface binds the default TCP and UDP dialers the same way [WithSource] binds the client's: an IP
// address or hostname sets their source address, and an interface name binds them to the device on Linux or sets
// the interface's address elsewhere. A network prefix such as "udp://" is ignored. Dialers set in the options are
// used as given. A SourceInterface that does not resolve makes every run of the analyzer fail.
func NewPacketLossAnalyzer(options *PacketLossAnalyzerOptions) *PacketLossAnalyzer {
	if options == nil {
		options = &PacketLossAnalyzerOptions{}
	}

	if options.SamplingDuration == 0 {
		options.SamplingDuration = time.Second * 30
	}

	if options.RemoteSamplingInterval == 0 {
		options.RemoteSamplingInterval = 1 * time.Second
	}

	if options.PacketSendingInterval == 0 {
		options.PacketSendingInterval = 67 * time.Millisecond
	}

	if options.PacketSendingTimeout == 0 {
		options.PacketSendingTimeout = defaultPacketSendingTimeout
	}

	source, sourceErr := resolveSource(options.SourceInterface)

	if options.TCPDialer == nil {
		options.TCPDialer = &net.Dialer{
			Timeout: options.PacketSendingTimeout,
			Control: source.control,
		}
		if source.sourceIP != nil {
			options.TCPDialer.LocalAddr = &net.TCPAddr{IP: source.sourceIP}
		}
	}

	if options.UDPDialer == nil {
		options.UDPDialer = &net.Dialer{
			Timeout: options.PacketSendingTimeout,
			Control: source.control,
		}
		if source.sourceIP != nil {
			options.UDPDialer.LocalAddr = &net.UDPAddr{IP: source.sourceIP}
		}
	}

	return &PacketLossAnalyzer{
		options:   options,
		sourceErr: sourceErr,
	}
}

// NewPacketLossAnalyzer creates a packet loss analyzer whose connections leave the way the client's do: from its
// source, through its socket control, and with its source-bound DNS resolver.
//
// Each dialer the options leave unset is a copy of the client's, with the analyzer's PacketSendingTimeout, so the
// client's own dialers never change. The copies make SourceInterface unnecessary. Other options keep the defaults
// of [NewPacketLossAnalyzer].
//
// Parameters:
//   - options: the analyzer options, or nil for the defaults.
//
// Returns:
//   - *PacketLossAnalyzer: the analyzer.
func (s *Client) NewPacketLossAnalyzer(options *PacketLossAnalyzerOptions) *PacketLossAnalyzer {
	if options == nil {
		options = &PacketLossAnalyzerOptions{}
	}

	if s == nil {
		return NewPacketLossAnalyzer(options)
	}

	timeout := options.PacketSendingTimeout
	if timeout == 0 {
		timeout = defaultPacketSendingTimeout
	}

	if options.TCPDialer == nil {
		options.TCPDialer = dialerWithTimeout(s.tcpDialer, timeout)
	}

	if options.UDPDialer == nil {
		options.UDPDialer = dialerWithTimeout(s.udpDialer, timeout)
	}

	return NewPacketLossAnalyzer(options)
}

// dialerWithTimeout returns a copy of dialer with a different connect timeout.
//
// Parameters:
//   - dialer: the dialer to copy, nil for a client that was not built by [New].
//   - timeout: the copy's connect timeout.
//
// Returns:
//   - *net.Dialer: the copy, or nil when dialer is nil, so [NewPacketLossAnalyzer] builds its default dialer.
func dialerWithTimeout(dialer *net.Dialer, timeout time.Duration) *net.Dialer {
	if dialer == nil {
		return nil
	}

	dialerCopy := *dialer
	dialerCopy.Timeout = timeout

	return &dialerCopy
}

// RunMulti Mix all servers to get the average packet loss.
func (pla *PacketLossAnalyzer) RunMulti(hosts []string) (*PacketLossStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pla.options.SamplingDuration)
	defer cancel()

	return pla.RunMultiWithContext(ctx, hosts)
}

// RunMultiWithContext runs packet loss analysis on multiple hosts concurrently with context.
//
// The result combines the last sample from each host that counted sent packets, as [combinePacketLoss] describes.
func (pla *PacketLossAnalyzer) RunMultiWithContext(
	ctx context.Context,
	hosts []string,
) (*PacketLossStats, error) {
	if pla.sourceErr != nil {
		return nil, fmt.Errorf(
			"source interface %q: %w",
			pla.options.SourceInterface,
			pla.sourceErr,
		)
	}

	results := make(map[string]*PacketLossStats)
	mutex := &sync.Mutex{}

	wg := &sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)

		go func(h string) {
			defer wg.Done()

			var lastValid *PacketLossStats

			_ = pla.RunWithContext(ctx, h, func(packetLoss *PacketLossStats) {
				if packetLoss.Sent != 0 {
					lastValid = packetLoss
				}
			})

			if lastValid != nil {
				mutex.Lock()
				results[h] = lastValid
				mutex.Unlock()
			}
		}(host)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, ErrUnsupported
	}

	return combinePacketLoss(results), nil
}

// Run performs packet loss analysis on a single host.
func (pla *PacketLossAnalyzer) Run(host string, callback func(packetLoss *PacketLossStats)) error {
	ctx, cancel := context.WithTimeout(context.Background(), pla.options.SamplingDuration)
	defer cancel()

	return pla.RunWithContext(ctx, host, callback)
}

// RunWithContext performs packet loss analysis on a single host with context.
//
// It connects a TCP sampler and a UDP sender, registers the sampler's UUID, sends datagrams, and reports the
// server's counts to callback at each sampling interval until ctx ends. Both connections are closed before it
// returns, after the send loop has stopped.
//
// Parameters:
//   - ctx: ends the analysis. Its end is the normal way to finish.
//   - host: the server's host:port.
//   - callback: receives each sample.
//
// Returns:
//   - error: the SourceInterface error when it did not resolve, [ErrUnsupported] wrapping the cause when
//     setup fails, the sampling error when the connection fails during the run, or nil when ctx ends the run.
func (pla *PacketLossAnalyzer) RunWithContext(
	ctx context.Context,
	host string,
	callback func(packetLoss *PacketLossStats),
) error {
	if pla.sourceErr != nil {
		return fmt.Errorf("source interface %q: %w", pla.options.SourceInterface, pla.sourceErr)
	}

	samplerClient, err := transport.NewClient(pla.options.TCPDialer)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	senderClient, err := transport.NewPacketLossSender(samplerClient.ID(), pla.options.UDPDialer)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	err = samplerClient.Connect(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	defer func() { _ = samplerClient.Disconnect() }()

	err = senderClient.Connect(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	defer func() { _ = senderClient.Close() }()

	err = samplerClient.InitPacketLossContext(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	// The send loop must stop before the sender is closed, so stop it and wait for it before the deferred Close.
	sendCtx, stopSending := context.WithCancel(ctx)

	var sending sync.WaitGroup

	sending.Go(func() { pla.loopSender(sendCtx, senderClient) })

	defer func() {
		stopSending()
		sending.Wait()
	}()

	return pla.loopSampler(ctx, samplerClient, callback)
}

// loopSampler reports the server's counts to callback at each sampling interval until ctx ends.
//
// Parameters:
//   - ctx: ends the sampling.
//   - client: the connected and registered sampler.
//   - callback: receives each sample.
//
// Returns:
//   - error: the first sampling error, or nil when ctx ends the sampling.
func (pla *PacketLossAnalyzer) loopSampler(
	ctx context.Context,
	client *transport.Client,
	callback func(packetLoss *PacketLossStats),
) error {
	ticker := time.NewTicker(pla.options.RemoteSamplingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			packetLoss, err := client.PacketLossContext(ctx)
			if err != nil {
				if endedByContext(ctx, err) {
					return nil
				}

				return fmt.Errorf("packet loss sampling failed: %w", err)
			}

			callback(
				&PacketLossStats{Sent: packetLoss.Sent, Dup: packetLoss.Dup, Max: packetLoss.Max},
			)
		case <-ctx.Done():
			return nil
		}
	}
}

// loopSender sends one numbered datagram per sending interval until ctx ends.
//
// Parameters:
//   - ctx: ends the sending.
//   - senderClient: the connected sender.
func (pla *PacketLossAnalyzer) loopSender(
	ctx context.Context,
	senderClient *transport.PacketLossSender,
) {
	order := 0

	sendTick := time.NewTicker(pla.options.PacketSendingInterval)
	defer sendTick.Stop()

	for {
		select {
		case <-sendTick.C:
			_ = senderClient.Send(order)
			order++
		case <-ctx.Done():
			return
		}
	}
}

// combinePacketLoss adds up the counts of several servers into one set with the same loss formula.
//
// Each server expected Max + 1 datagrams, so the combined Max is the sum of those, less one. Adding the Max values
// alone would lose one datagram from the denominator for every server after the first.
//
// Parameters:
//   - results: each server's counts, keyed by host.
//
// Returns:
//   - *PacketLossStats: the combined counts.
func combinePacketLoss(results map[string]*PacketLossStats) *PacketLossStats {
	var combined PacketLossStats

	for _, hostPacketLoss := range results {
		combined.Sent += hostPacketLoss.Sent
		combined.Dup += hostPacketLoss.Dup
		combined.Max += hostPacketLoss.Max + 1
	}

	combined.Max--

	return &combined
}

// endedByContext reports whether a sampling error means the run's context ended rather than the connection failing.
//
// The sampler's connection deadline equals the context deadline, so the request can fail with a deadline error a
// moment before ctx.Err reports it. The transport attributes such errors to the context, so they count as well.
//
// Parameters:
//   - ctx: the run's context.
//   - err: the sampling error.
//
// Returns:
//   - bool: true when the error marks the normal end of the run.
func endedByContext(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// PacketLoss measures the packet loss to a server with the speedtest.net TCP and UDP protocols until ctx ends. The
// connections leave the way the client's others do, as with [Client.NewPacketLossAnalyzer].
//
// Parameters:
//   - ctx: ends the measurement. Its end is the normal way to finish.
//   - server: the server, whose host serves the protocols.
//
// Returns:
//   - PacketLossStats: the last sample in which the server counted sent packets, or zero counts when none did. It is
//     also returned with an error, so a measurement that failed partway keeps what it saw.
//   - error: [ErrServerNil], [ErrUninitializedManager] for a client not built by [New], an error wrapping
//     [ErrUnsupported] when the server cannot be reached, or the sampling error.
func (s *Client) PacketLoss(ctx context.Context, server *Server) (PacketLossStats, error) {
	if !s.ready() {
		return PacketLossStats{}, ErrUninitializedManager
	}

	if server == nil {
		return PacketLossStats{}, ErrServerNil
	}

	host, err := tcpHost(server)
	if err != nil {
		return PacketLossStats{}, fmt.Errorf("%w: %w", ErrUnsupported, err)
	}

	var last PacketLossStats

	err = s.NewPacketLossAnalyzer(nil).RunWithContext(ctx, host, func(packetLoss *PacketLossStats) {
		if packetLoss.Sent != 0 {
			last = *packetLoss
		}
	})
	if err != nil {
		return last, fmt.Errorf("packet loss measurement failed: %w", err)
	}

	return last, nil
}

// String formats the packet loss as a percentage with the counts, or reports it unavailable when the server
// received nothing, such as when a gateway drops every datagram.
//
// Returns:
//   - string: the formatted packet loss.
func (p PacketLossStats) String() string {
	if p.Sent == 0 {
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
//
// Counts that do not fit together, such as more unique datagrams than were sent, are held to the range 0 to 1, so
// -1 always means that the server received nothing. A Max below 0 means that no datagram was expected, so nothing
// counts as lost.
//
// Returns:
//   - float64: the ratio from 0 to 1, or -1 when the server received nothing.
func (p PacketLossStats) Loss() float64 {
	if p.Sent == 0 {
		return -1
	}

	expected := p.Max + 1
	if expected <= 0 {
		return 0
	}

	return min(max(1-float64(p.Sent-p.Dup)/float64(expected), 0), 1)
}

// LossPercent returns the packet loss percentage.
//
// Returns:
//   - float64: the percentage from 0 to 100, or -1 when the server received nothing.
func (p PacketLossStats) LossPercent() float64 {
	if p.Sent == 0 {
		return -1
	}

	return p.Loss() * 100
}

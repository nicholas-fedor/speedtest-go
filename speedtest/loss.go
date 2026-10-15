package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
)

// PacketLossAnalyzerOptions configures the packet loss analyzer.
type PacketLossAnalyzerOptions struct {
	RemoteSamplingInterval time.Duration
	SamplingDuration       time.Duration
	PacketSendingInterval  time.Duration
	PacketSendingTimeout   time.Duration
	SourceInterface        string      // source interface
	TCPDialer              *net.Dialer // tcp dialer for sampling
	UDPDialer              *net.Dialer // udp dialer for sending packet
}

// defaultPacketSendingTimeout bounds connecting the analyzer's sockets when the options set no timeout.
const defaultPacketSendingTimeout = 5 * time.Second

// PacketLossAnalyzer performs packet loss analysis on network connections.
type PacketLossAnalyzer struct {
	options *PacketLossAnalyzerOptions
	// sourceErr is why SourceInterface did not resolve, returned by every run, or nil.
	sourceErr error
}

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
func (pla *PacketLossAnalyzer) RunMulti(hosts []string) (*transport.PLoss, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pla.options.SamplingDuration)
	defer cancel()

	return pla.RunMultiWithContext(ctx, hosts)
}

// RunMultiWithContext runs packet loss analysis on multiple hosts concurrently with context.
func (pla *PacketLossAnalyzer) RunMultiWithContext(
	ctx context.Context,
	hosts []string,
) (*transport.PLoss, error) {
	if pla.sourceErr != nil {
		return nil, fmt.Errorf(
			"source interface %q: %w",
			pla.options.SourceInterface,
			pla.sourceErr,
		)
	}

	results := make(map[string]*transport.PLoss)
	mutex := &sync.Mutex{}

	wg := &sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)

		go func(h string) {
			defer wg.Done()

			var lastValid *transport.PLoss

			_ = pla.RunWithContext(ctx, h, func(packetLoss *transport.PLoss) {
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
		return nil, transport.ErrUnsupported
	}

	var pLoss transport.PLoss
	for _, hostPacketLoss := range results {
		pLoss.Sent += hostPacketLoss.Sent
		pLoss.Dup += hostPacketLoss.Dup
		pLoss.Max += hostPacketLoss.Max
	}

	return &pLoss, nil
}

// Run performs packet loss analysis on a single host.
func (pla *PacketLossAnalyzer) Run(host string, callback func(packetLoss *transport.PLoss)) error {
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
//   - error: the SourceInterface error when it did not resolve, [transport.ErrUnsupported] wrapping the cause when
//     setup fails, the sampling error when the connection fails during the run, or nil when ctx ends the run.
func (pla *PacketLossAnalyzer) RunWithContext(
	ctx context.Context,
	host string,
	callback func(packetLoss *transport.PLoss),
) error {
	if pla.sourceErr != nil {
		return fmt.Errorf("source interface %q: %w", pla.options.SourceInterface, pla.sourceErr)
	}

	samplerClient, err := transport.NewClient(pla.options.TCPDialer)
	if err != nil {
		return fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
	}

	senderClient, err := transport.NewPacketLossSender(samplerClient.ID(), pla.options.UDPDialer)
	if err != nil {
		return fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
	}

	err = samplerClient.Connect(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
	}

	defer func() { _ = samplerClient.Disconnect() }()

	err = senderClient.Connect(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
	}

	defer func() { _ = senderClient.Close() }()

	err = samplerClient.InitPacketLossContext(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
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
	callback func(packetLoss *transport.PLoss),
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

			callback(packetLoss)
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
//   - transport.PLoss: the last sample in which the server counted sent packets, or zero counts when none did. It is
//     also returned with an error, so a measurement that failed partway keeps what it saw.
//   - error: [ErrServerNil], [ErrUninitializedManager] for a client not built by [New], an error wrapping
//     [transport.ErrUnsupported] when the server cannot be reached, or the sampling error.
func (s *Client) PacketLoss(ctx context.Context, server *Server) (transport.PLoss, error) {
	if !s.ready() {
		return transport.PLoss{}, ErrUninitializedManager
	}

	if server == nil {
		return transport.PLoss{}, ErrServerNil
	}

	host, err := tcpHost(server)
	if err != nil {
		return transport.PLoss{}, fmt.Errorf("%w: %w", transport.ErrUnsupported, err)
	}

	var last transport.PLoss

	err = s.NewPacketLossAnalyzer(nil).RunWithContext(ctx, host, func(packetLoss *transport.PLoss) {
		if packetLoss.Sent != 0 {
			last = *packetLoss
		}
	})
	if err != nil {
		return last, fmt.Errorf("packet loss measurement failed: %w", err)
	}

	return last, nil
}

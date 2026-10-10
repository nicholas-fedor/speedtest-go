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

// PacketLossAnalyzer performs packet loss analysis on network connections.
type PacketLossAnalyzer struct {
	options *PacketLossAnalyzerOptions
}

// NewPacketLossAnalyzer creates a new packet loss analyzer with the given options.
//
// When SourceInterface is an IP address, optionally with a network prefix such as "udp://", the default TCP and
// UDP dialers bind to it. Other values leave the dialers unbound. Dialers set in the options are used as given.
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
		options.PacketSendingTimeout = 5 * time.Second
	}

	sourceIP := packetLossSourceIP(options.SourceInterface)

	if options.TCPDialer == nil {
		options.TCPDialer = &net.Dialer{Timeout: options.PacketSendingTimeout}
		if sourceIP != nil {
			options.TCPDialer.LocalAddr = &net.TCPAddr{IP: sourceIP}
		}
	}

	if options.UDPDialer == nil {
		options.UDPDialer = &net.Dialer{Timeout: options.PacketSendingTimeout}
		if sourceIP != nil {
			options.UDPDialer.LocalAddr = &net.UDPAddr{IP: sourceIP}
		}
	}

	return &PacketLossAnalyzer{
		options: options,
	}
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
//   - error: [transport.ErrUnsupported] wrapping the cause when setup fails, the sampling error when the
//     connection fails during the run, or nil when ctx ends the run.
func (pla *PacketLossAnalyzer) RunWithContext(
	ctx context.Context,
	host string,
	callback func(packetLoss *transport.PLoss),
) error {
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

// packetLossSourceIP parses the analyzer's source as an IP address.
//
// Parameters:
//   - source: the configured source, possibly with a network prefix such as "udp://".
//
// Returns:
//   - net.IP: the source IP, or nil when the source is empty or not an IP address.
func packetLossSourceIP(source string) net.IP {
	if len(source) == 0 {
		return nil
	}

	_, address := parseAddr(source)

	return net.ParseIP(address)
}

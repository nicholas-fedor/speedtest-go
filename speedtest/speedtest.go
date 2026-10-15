package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/meter"
)

// modulePath is this module's path; used to resolve version from build info.
const modulePath = "github.com/nicholas-fedor/speedtest-go/v2"

// version is set via ldflags; empty means resolve from build info.
var version string

// Proto represents the protocol type for ping operations.
type Proto int

const (
	// HTTP is the HTTP protocol.
	HTTP Proto = iota
	// TCP is the TCP protocol.
	TCP
	// ICMP is the ICMP protocol.
	ICMP
)

// TestMode selects the protocol that download and upload tests use.
type TestMode int

// Test modes.
const (
	// HTTPTest downloads files and posts uploads over HTTP. It is the default.
	HTTPTest TestMode = iota
	// TCPTest uses the speedtest.net TCP protocol on the server's TCP port, normally 8080. HTTP proxies do not
	// apply to it.
	TCPTest
)

// Client is a speedtest client. [New] builds one from options.
type Client struct {
	// meter runs the transfer workers of download and upload tests and keeps their totals.
	meter *meter.Meter
	// doer sends HTTP requests.
	doer *http.Client
	// transport is the client's HTTP transport, built from its settings.
	transport *http.Transport
	// tcpDialer dials TCP connections, including those of the transport.
	tcpDialer *net.Dialer
	// udpDialer dials the UDP sockets of packet loss tests.
	udpDialer *net.Dialer
	// ipDialer dials ICMP sockets.
	ipDialer *net.Dialer
	// settings is the configuration the client was built from.
	settings settings
}

// Network timeouts used by the dialers and the HTTP transport.
const (
	// dialTimeout bounds establishing a TCP connection or an ICMP socket.
	dialTimeout = 30 * time.Second
	// dialKeepAlive is the TCP keep-alive period.
	dialKeepAlive = 30 * time.Second
	// dnsDialTimeout bounds connecting to a DNS server through the source-bound resolver.
	dnsDialTimeout = 5 * time.Second
	// idleConnTimeout closes idle HTTP connections.
	idleConnTimeout = 90 * time.Second
	// tlsHandshakeTimeout bounds TLS handshakes.
	tlsHandshakeTimeout = 10 * time.Second
	// expectContinueTimeout bounds waiting for a 100 Continue response.
	expectContinueTimeout = 1 * time.Second
	// maxIdleConns caps idle HTTP connections across all hosts.
	maxIdleConns = 100
)

func parseAddr(addr string) (string, string) {
	before, after, ok := strings.Cut(addr, "://")
	if ok {
		return before, after
	}

	return "", addr // ignore address network prefix
}

// userAgentTransport sends requests with a fixed User-Agent header.
type userAgentTransport struct {
	// base sends the requests.
	base http.RoundTripper
	// userAgent is the header value.
	userAgent string
}

var _ http.RoundTripper = (*userAgentTransport)(nil)

// RoundTrip sends a copy of the request with the User-Agent header set, so the caller's request is never modified
// and the header is never sent twice.
//
// Parameters:
//   - req: the request.
//
// Returns:
//   - *http.Response: the response.
//   - error: the error of the base transport.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	out.Header.Set("User-Agent", t.userAgent)

	resp, err := t.base.RoundTrip(out)
	if err != nil {
		return nil, fmt.Errorf("failed to round trip request: %w", err)
	}

	return resp, nil
}

// New creates a speedtest client.
//
// Every option is validated, and an error lists each invalid one. The client owns its HTTP client, dialers, and
// transport, so it never shares or modifies [http.DefaultClient] or [net.DefaultResolver].
//
// Parameters:
//   - opts: the options, applied in order.
//
// Returns:
//   - *Client: the client.
//   - error: an error wrapping [ErrInvalidOption] for each invalid option.
func New(opts ...Option) (*Client, error) {
	cfg := defaultSettings()

	var errs []error

	for _, opt := range opts {
		if opt == nil {
			errs = append(errs, fmt.Errorf("%w: the option is nil", ErrInvalidOption))

			continue
		}

		err := opt(&cfg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	client := newClient(cfg)
	client.logSettings()

	return client, nil
}

// defaultSettings returns the settings of a client built without options.
//
// Returns:
//   - settings: the default settings.
func defaultSettings() settings {
	return settings{
		userAgent: DefaultUserAgent(),
		baseURL:   DefaultBaseURL,
	}
}

// newClient builds a client from validated settings.
//
// It always builds the TCP, UDP, and ICMP dialers and the HTTP transport, so the user agent, proxy, source, DSCP
// marking, and dialer controls apply whether or not a source is set.
//
// Parameters:
//   - cfg: the settings, which the options have validated.
//
// Returns:
//   - *Client: the client.
func newClient(cfg settings) *Client {
	controls := append([]dialControl{cfg.source.control, dscpControl(cfg.dscp)}, cfg.controls...)
	control := chainControl(controls...)

	var resolver *net.Resolver
	if cfg.dnsBindSource && (cfg.source.sourceIP != nil || control != nil) {
		resolver = sourceBoundResolver(cfg.source.sourceIP, control)
	}

	client := &Client{
		meter: meter.New(meter.Config{
			Logger:               cfg.logger,
			Connections:          cfg.connections,
			CaptureTime:          cfg.testDuration,
			RateCaptureFrequency: 0,
		}),
		settings: cfg,
		tcpDialer: &net.Dialer{
			LocalAddr: cfg.source.tcp,
			Timeout:   dialTimeout,
			KeepAlive: dialKeepAlive,
			Control:   control,
			Resolver:  resolver,
		},
		udpDialer: &net.Dialer{
			Timeout:  dialTimeout,
			Control:  control,
			Resolver: resolver,
		},
		ipDialer: &net.Dialer{
			LocalAddr: cfg.source.ip,
			Timeout:   dialTimeout,
			KeepAlive: dialKeepAlive,
			Control:   control,
			Resolver:  resolver,
		},
	}

	if cfg.source.sourceIP != nil {
		client.udpDialer.LocalAddr = &net.UDPAddr{IP: cfg.source.sourceIP}
	}

	proxy := http.ProxyFromEnvironment
	if cfg.proxy != nil {
		proxy = http.ProxyURL(cfg.proxy)
	}

	client.transport = &http.Transport{
		Proxy:                 proxy,
		DialContext:           client.tcpDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}

	ownTransport := &userAgentTransport{base: client.transport, userAgent: cfg.userAgent}

	client.doer = &http.Client{Transport: ownTransport}
	if cfg.httpClient != nil {
		client.doer = cfg.httpClient
		if client.doer.Transport == nil {
			client.doer.Transport = ownTransport
		}
	}

	return client
}

// sourceBoundResolver returns a DNS resolver whose queries leave from the source.
//
// It uses the pure Go resolver, because the cgo resolver ignores a custom Dial function.
//
// Parameters:
//   - sourceIP: the local IP to send DNS queries from, or nil when control binds them to an interface instead.
//   - control: the client's socket control, including any binding to the source interface, or nil.
//
// Returns:
//   - *net.Resolver: the resolver for the client's dialers.
func sourceBoundResolver(sourceIP net.IP, control dialControl) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, dnsServer string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout: dnsDialTimeout,
				Control: control,
			}

			if sourceIP != nil {
				dialer.LocalAddr = localAddrFor(network, sourceIP)
			}

			return dialer.DialContext(ctx, network, dnsServer)
		},
	}
}

// localAddrFor returns a local address of the right type for a DNS dial on network.
//
// Parameters:
//   - network: the network the resolver dials, such as "udp" or "tcp4".
//   - ip: the local IP.
//
// Returns:
//   - net.Addr: a UDP or TCP address for ip, or nil for other networks.
func localAddrFor(network string, ip net.IP) net.Addr {
	switch network {
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip}
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip}
	default:
		return nil
	}
}

// DefaultUserAgent returns the default user agent string for speedtest requests.
//
// Returns:
//   - string: user agent including the resolved library version
func DefaultUserAgent() string {
	return "nicholas-fedor/speedtest-go " + Version()
}

// Version returns the version of the speedtest library.
//
// Resolution order: ldflag version, then this module's version from
// debug.BuildInfo (Main or Deps), then "dev".
//
// Returns:
//   - string: normalized version without a leading v
func Version() string {
	buildInfo, ok := debug.ReadBuildInfo()

	return resolveVersion(version, buildInfo, ok)
}

// resolveVersion picks the library version from an ldflag value and build info.
//
// It reads no package state, so tests can cover every resolution path in parallel.
//
// Parameters:
//   - ldflag: the version injected at link time, possibly empty.
//   - buildInfo: the binary's build info, ignored when ok is false.
//   - ok: whether buildInfo is available.
//
// Returns:
//   - string: normalized version without a leading v, or "dev" when unresolved.
func resolveVersion(ldflag string, buildInfo *debug.BuildInfo, ok bool) string {
	if resolved := normalizeVersion(ldflag); resolved != "" {
		return resolved
	}

	if !ok || buildInfo == nil {
		return "dev"
	}

	if buildInfo.Main.Path == modulePath {
		if resolved := usableModuleVersion(buildInfo.Main.Version); resolved != "" {
			return resolved
		}
	}

	for _, dep := range buildInfo.Deps {
		if dep.Path == modulePath {
			if resolved := usableModuleVersion(dep.Version); resolved != "" {
				return resolved
			}
		}
	}

	return "dev"
}

// normalizeVersion trims a single leading v from a version token.
func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "v")
}

// usableModuleVersion returns a normalized module version, or empty if unusable.
func usableModuleVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "(devel)" {
		return ""
	}

	return normalizeVersion(raw)
}

// logSettings records the client's settings as a debug message.
func (s *Client) logSettings() {
	proxy := ""
	if s.settings.proxy != nil {
		proxy = s.settings.proxy.Redacted()
	}

	s.log().Debug(
		context.Background(),
		"client settings",
		logKeySource, s.settings.sourceName,
		logKeyProxy, proxy,
		logKeyBaseURL, s.settings.baseURL,
		logKeyConnections, s.settings.connections,
		logKeySearch, s.settings.search,
		logKeyPingMode, s.settings.pingMode,
		logKeyProtocol, s.settings.testMode,
		logKeyOS, runtime.GOOS,
		logKeyArch, runtime.GOARCH,
		logKeyCPUs, runtime.NumCPU(),
	)
}

// log returns the client's logger, which discards messages when none is set.
//
// Returns:
//   - Logger: the logger.
func (s *Client) log() Logger {
	if s == nil {
		return noopLogger{}
	}

	return loggerOrNoop(s.settings.logger)
}

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"
)

// modulePath is this module's path; used to resolve version from build info.
const modulePath = "github.com/nicholas-fedor/speedtest-go/v2"

// version is set via ldflags; empty means resolve from build info.
var version string

var (
	// ErrClientNil is returned when the speedtest client is nil.
	ErrClientNil = errors.New("speedtest client is nil")
	// ErrRequestNil is returned when the request is nil.
	ErrRequestNil = errors.New("request is nil")
)

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

// Speedtest is a speedtest client.
type Speedtest struct {
	Manager

	userMu sync.RWMutex
	User   *User

	doer      *http.Client
	config    *UserConfig
	tcpDialer *net.Dialer
	udpDialer *net.Dialer
	ipDialer  *net.Dialer
}

// UserConfig holds configuration options for speedtest.
type UserConfig struct {
	// T is the HTTP transport built from this configuration. NewUserConfig replaces any value set here.
	T         *http.Transport
	UserAgent string
	Proxy     string
	// Source is where connections leave from: an IP address, a network interface name, or a hostname, optionally
	// with a network prefix such as "tcp://". On Linux, an interface name binds each socket to the interface, which
	// then supplies an address of the destination's family. Elsewhere it uses the address [ResolveInterface] picks.
	Source        string
	DNSBindSource bool
	// DialerControl runs on each socket the client dials before it connects, after any binding to a source
	// interface and any DSCP marking. With DNSBindSource, it also runs on DNS sockets.
	DialerControl func(network, address string, c syscall.RawConn) error
	// DSCP marks each socket the client dials with a Differentiated Services code point from 0 to 63, where 0 leaves
	// sockets unmarked. A value that [CheckDSCP] rejects is ignored.
	DSCP     int
	Debug    bool
	PingMode Proto
	// TestMode selects the protocol for download and upload tests. The ping method is set by PingMode.
	TestMode TestMode
	// DownloadSize selects the download block size from 350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, or 4000.
	// Over HTTP it is the side of the requested image, and over TCP the size in kB. 0 keeps the default of 1000, and
	// a value that [CheckDownloadSize] rejects is ignored.
	DownloadSize int
	// UploadSize selects the upload block size in kB from 100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, or 4000.
	// 0 keeps the defaults of 1000 for single-server and 800 for multi-server tests, and a value that
	// [CheckUploadSize] rejects is ignored.
	UploadSize int

	SavingMode     bool
	MaxConnections int

	CityFlag     string
	LocationFlag string
	Location     *Location

	Keyword string // Fuzzy search

	// BaseURL is the speedtest API base URL. Empty means [DefaultBaseURL].
	BaseURL string
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

// NewUserConfig sets the user configuration for the speedtest instance.
//
// It always builds the TCP, UDP, and ICMP dialers and the HTTP transport from the configuration, so the user agent,
// proxy, source address, and dialer control apply whether or not a source address is set. The client's own HTTP
// client sends requests through that transport. Process-wide defaults such as [http.DefaultClient] and
// [net.DefaultResolver] are never modified.
func (s *Speedtest) NewUserConfig(userConfig *UserConfig) {
	if userConfig.Debug {
		dbg.Enable()
	}

	if userConfig.SavingMode {
		userConfig.MaxConnections = 1 // Set the number of concurrent connections to 1
	}

	s.SetNThread(userConfig.MaxConnections)

	if len(userConfig.CityFlag) > 0 {
		var err error

		userConfig.Location, err = GetLocation(userConfig.CityFlag)
		if err != nil {
			dbg.Printf("Warning: skipping command line arguments: --city. err: %v\n", err.Error())
		}
	}

	if len(userConfig.LocationFlag) > 0 {
		var err error

		userConfig.Location, err = ParseLocation(userConfig.CityFlag, userConfig.LocationFlag)
		if err != nil {
			dbg.Printf(
				"Warning: skipping command line arguments: --location. err: %v\n",
				err.Error(),
			)
		}
	}

	s.config = userConfig
	if len(s.config.UserAgent) == 0 {
		s.config.UserAgent = DefaultUserAgent()
	}

	if len(s.config.BaseURL) == 0 {
		s.config.BaseURL = DefaultBaseURL
	}

	err := CheckDownloadSize(s.config.DownloadSize)
	if err != nil {
		dbg.Printf("Warning: skipping the download size. err: %s\n", err.Error())

		s.config.DownloadSize = 0
	}

	err = CheckUploadSize(s.config.UploadSize)
	if err != nil {
		dbg.Printf("Warning: skipping the upload size. err: %s\n", err.Error())

		s.config.UploadSize = 0
	}

	source := resolveSource(userConfig.Source)
	control := chainControl(source.control, dscpControl(userConfig.DSCP), userConfig.DialerControl)

	var resolver *net.Resolver
	if userConfig.DNSBindSource && (source.sourceIP != nil || control != nil) {
		resolver = sourceBoundResolver(source.sourceIP, control)
	}

	s.tcpDialer = &net.Dialer{
		LocalAddr: source.tcp,
		Timeout:   dialTimeout,
		KeepAlive: dialKeepAlive,
		Control:   control,
		Resolver:  resolver,
	}

	s.udpDialer = &net.Dialer{
		Timeout:  dialTimeout,
		Control:  control,
		Resolver: resolver,
	}
	if source.sourceIP != nil {
		s.udpDialer.LocalAddr = &net.UDPAddr{IP: source.sourceIP}
	}

	s.ipDialer = &net.Dialer{
		LocalAddr: source.ip,
		Timeout:   dialTimeout,
		KeepAlive: dialKeepAlive,
		Control:   control,
		Resolver:  resolver,
	}

	s.config.T = &http.Transport{
		Proxy:                 proxyFunc(userConfig.Proxy),
		DialContext:           s.tcpDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}
}

// RoundTrip executes a single HTTP request using the speedtest client's round tripper.
//
// It sends a copy of req with the configured User-Agent, so the caller's request is never modified and the header
// is never sent twice.
func (s *Speedtest) RoundTrip(req *http.Request) (*http.Response, error) {
	if s == nil {
		return nil, ErrClientNil
	}

	if req == nil {
		return nil, ErrRequestNil
	}

	out := req.Clone(req.Context())
	out.Header.Set("User-Agent", s.config.UserAgent)

	resp, err := s.config.T.RoundTrip(out)
	if err != nil {
		return nil, fmt.Errorf("failed to round trip request: %w", err)
	}

	return resp, nil
}

// Option is a function that can be passed to New to modify the Client.
type Option func(*Speedtest)

// WithDoer sets the http.Client used to make requests.
//
// The client is copied, so the caller's client is never modified. When the copy has no Transport, requests go
// through the speedtest client's own round tripper, which applies the user agent, proxy, and dialers from the user
// configuration. A copy with its own Transport keeps it and bypasses those settings. A nil client is ignored.
func WithDoer(doer *http.Client) Option {
	return func(s *Speedtest) {
		if doer == nil {
			return
		}

		client := *doer
		if client.Transport == nil {
			client.Transport = s
		}

		s.doer = &client
	}
}

// WithUserConfig adds a custom user config for speedtest.
//
// The configuration applies to requests sent through the speedtest client's round tripper, which includes the
// default HTTP client and any client given to [WithDoer] without its own Transport.
func WithUserConfig(userConfig *UserConfig) Option {
	return func(s *Speedtest) {
		s.NewUserConfig(userConfig)
		dbg.Printf("Source: %s\n", s.config.Source)
		dbg.Printf("Proxy: %s\n", s.config.Proxy)
		dbg.Printf("BaseURL: %s\n", s.config.BaseURL)
		dbg.Printf("SavingMode: %v\n", s.config.SavingMode)
		dbg.Printf("Keyword: %v\n", s.config.Keyword)
		dbg.Printf("PingType: %v\n", s.config.PingMode)
		dbg.Printf("OS: %s, ARCH: %s, NumCPU: %d\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}
}

// New creates a new speedtest client.
//
// The client owns its HTTP client, so it never shares or modifies [http.DefaultClient].
func New(opts ...Option) *Speedtest {
	s := &Speedtest{
		Manager: NewDataManager(),
	}
	s.doer = &http.Client{Transport: s}

	// load default config
	s.NewUserConfig(&UserConfig{})

	for _, opt := range opts {
		opt(s)
	}

	return s
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

// proxyFunc returns the proxy selector for the HTTP transport.
//
// Parameters:
//   - proxy: the configured proxy URL, possibly empty.
//
// Returns:
//   - func(*http.Request) (*url.URL, error): a selector that always uses the configured proxy, or
//     [http.ProxyFromEnvironment] when none is set or it does not parse.
func proxyFunc(proxy string) func(*http.Request) (*url.URL, error) {
	if len(proxy) == 0 {
		return http.ProxyFromEnvironment
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		dbg.Printf("Warning: skipping parse the proxy host. err: %s\n", err.Error())

		return http.ProxyFromEnvironment
	}

	return http.ProxyURL(proxyURL)
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

var defaultClient = New()

// GetUser returns the user information in a thread-safe manner.
func (s *Speedtest) GetUser() *User {
	s.userMu.RLock()
	defer s.userMu.RUnlock()

	return s.User
}

// SetUser sets the user information in a thread-safe manner.
func (s *Speedtest) SetUser(u *User) {
	s.userMu.Lock()
	defer s.userMu.Unlock()

	s.User = u
}

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var (
	version = "1.7.10"
	// DefaultUserAgent is the default user agent string for speedtest requests.
	DefaultUserAgent = "showwin/speedtest-go " + version
)

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

// Speedtest is a speedtest client.
type Speedtest struct {
	Manager

	User *User

	doer      *http.Client
	config    *UserConfig
	tcpDialer *net.Dialer
	ipDialer  *net.Dialer
}

// UserConfig holds configuration options for speedtest.
type UserConfig struct {
	T             *http.Transport
	UserAgent     string
	Proxy         string
	Source        string
	DNSBindSource bool
	DialerControl func(network, address string, c syscall.RawConn) error
	Debug         bool
	PingMode      Proto

	SavingMode     bool
	MaxConnections int

	CityFlag     string
	LocationFlag string
	Location     *Location

	Keyword string // Fuzzy search
}

func parseAddr(addr string) (string, string) {
	before, after, ok := strings.Cut(addr, "://")
	if ok {
		return before, after
	}

	return "", addr // ignore address network prefix
}

// NewUserConfig sets the user configuration for the speedtest instance.
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

	var tcpSource net.Addr // If nil, a local address is automatically chosen.

	var (
		icmpSource net.Addr
		proxy      = http.ProxyFromEnvironment
	)

	s.config = userConfig
	if len(s.config.UserAgent) == 0 {
		s.config.UserAgent = DefaultUserAgent
	}

	if len(userConfig.Source) == 0 {
		return
	}

	_, address := parseAddr(userConfig.Source)

	addr0, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("[%s]:0", address))
	if err == nil {
		tcpSource = addr0
	} else {
		dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
	}

	addr1, err := net.ResolveIPAddr("ip", address)
	if err == nil {
		icmpSource = addr1
	} else {
		dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
	}

	if !userConfig.DNSBindSource {
		return
	}

	net.DefaultResolver.Dial = func(ctx context.Context, network, dnsServer string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout: 5 * time.Second,
			LocalAddr: func(network string) net.Addr {
				switch network {
				case "udp", "udp4", "udp6":
					return &net.UDPAddr{IP: net.ParseIP(address)}
				case "tcp", "tcp4", "tcp6":
					return &net.TCPAddr{IP: net.ParseIP(address)}
				default:
					return nil
				}
			}(network),
		}

		return dialer.DialContext(ctx, network, dnsServer)
	}

	if len(userConfig.Proxy) > 0 {
		parse, err := url.Parse(userConfig.Proxy)
		if err != nil {
			dbg.Printf("Warning: skipping parse the proxy host. err: %s\n", err.Error())
		} else {
			proxy = func(_ *http.Request) (*url.URL, error) {
				return parse, nil
			}
		}
	}

	s.tcpDialer = &net.Dialer{
		LocalAddr: tcpSource,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   userConfig.DialerControl,
	}

	s.ipDialer = &net.Dialer{
		LocalAddr: icmpSource,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   userConfig.DialerControl,
	}

	s.config.T = &http.Transport{
		Proxy:                 proxy,
		DialContext:           s.tcpDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	s.doer.Transport = s
}

// RoundTrip executes a single HTTP request using the speedtest client's round tripper.
func (s *Speedtest) RoundTrip(req *http.Request) (*http.Response, error) {
	if s == nil {
		return nil, ErrClientNil
	}

	if req == nil {
		return nil, ErrRequestNil
	}

	req.Header.Add("User-Agent", s.config.UserAgent)

	resp, err := s.config.T.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("failed to round trip request: %w", err)
	}

	return resp, nil
}

// Option is a function that can be passed to New to modify the Client.
type Option func(*Speedtest)

// WithDoer sets the http.Client used to make requests.
func WithDoer(doer *http.Client) Option {
	return func(s *Speedtest) {
		s.doer = doer
	}
}

// WithUserConfig adds a custom user config for speedtest.
// This configuration may be overwritten again by WithDoer,
// because client and transport are parent-child relationship:
// `New(WithDoer(myDoer), WithUserAgent(myUserAgent), WithDoer(myDoer))`.
func WithUserConfig(userConfig *UserConfig) Option {
	return func(s *Speedtest) {
		s.NewUserConfig(userConfig)
		dbg.Printf("Source: %s\n", s.config.Source)
		dbg.Printf("Proxy: %s\n", s.config.Proxy)
		dbg.Printf("SavingMode: %v\n", s.config.SavingMode)
		dbg.Printf("Keyword: %v\n", s.config.Keyword)
		dbg.Printf("PingType: %v\n", s.config.PingMode)
		dbg.Printf("OS: %s, ARCH: %s, NumCPU: %d\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}
}

// New creates a new speedtest client.
func New(opts ...Option) *Speedtest {
	s := &Speedtest{
		doer:    http.DefaultClient,
		Manager: NewDataManager(),
	}
	// load default config
	s.NewUserConfig(&UserConfig{UserAgent: DefaultUserAgent})

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Version returns the version of the speedtest library.
func Version() string {
	return version
}

var defaultClient = New()

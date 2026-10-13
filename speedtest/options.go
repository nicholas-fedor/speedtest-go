// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"syscall"
	"time"
)

// ErrInvalidOption is returned by [New] for an option with a value the client cannot use.
var ErrInvalidOption = errors.New("invalid option")

// proxySchemes are the proxy URL schemes that [http.Transport] supports.
var proxySchemes = []string{"http", "https", "socks5", "socks5h"}

// Option configures a [Client] that [New] builds.
//
// Options validate their values when New applies them, and New reports every invalid option at once. The order of
// options does not matter, except that a later option replaces an earlier one that sets the same value.
type Option func(*settings) error

// settings is a client's configuration, collected from its options before [New] builds the client.
type settings struct {
	// logger receives debug messages, or nil to discard them.
	logger Logger
	// httpClient is a copy of the caller's HTTP client from [WithHTTPClient], or nil for one of the client's own.
	httpClient *http.Client
	// location ranks servers by distance from it, or nil for the location speedtest.net reports.
	location *Location
	// proxy is the HTTP proxy, or nil to use the one the environment sets.
	proxy *url.URL
	// userAgent is the User-Agent header of every HTTP request.
	userAgent string
	// sourceName is the source as the caller gave it, for debug messages.
	sourceName string
	// baseURL is the speedtest API base URL.
	baseURL string
	// search filters the server list by keyword, or is empty for no filter.
	search string
	// source is where connections leave from, resolved by [WithSource].
	source sourceBinding
	// controls run on each socket the client dials, in the order [WithDialerControl] added them.
	controls []dialControl
	// dscp marks each socket with a Differentiated Services code point, where 0 leaves sockets unmarked.
	dscp int
	// downloadSize is the download block size, or 0 for the default.
	downloadSize int
	// uploadSize is the upload block size in kB, or 0 for the defaults.
	uploadSize int
	// connections is the number of connections a transfer test uses, or 0 for one per CPU.
	connections int
	// testDuration is how long each transfer test runs, or 0 for the default.
	testDuration time.Duration
	// pingMode is the method used to measure latency.
	pingMode Proto
	// testMode is the protocol used for download and upload tests.
	testMode TestMode
	// dnsBindSource sends DNS queries from the source too.
	dnsBindSource bool
}

// WithHTTPClient sets the HTTP client that sends the client's requests.
//
// The client is copied, so the caller's client is never modified. When it has no Transport, requests go through the
// client's own transport, which applies the user agent, proxy, source, DSCP marking, and dialer controls. A client
// with its own Transport keeps it, and those settings then do not apply to HTTP requests.
//
// Parameters:
//   - client: the HTTP client.
//
// Returns:
//   - Option: the option, which fails for a nil client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *settings) error {
		if client == nil {
			return fmt.Errorf("%w: the HTTP client is nil", ErrInvalidOption)
		}

		copied := *client
		cfg.httpClient = &copied

		return nil
	}
}

// WithUserAgent sets the User-Agent header of every HTTP request, in place of [DefaultUserAgent].
//
// Parameters:
//   - userAgent: the header value.
//
// Returns:
//   - Option: the option, which fails for an empty value.
func WithUserAgent(userAgent string) Option {
	return func(cfg *settings) error {
		if userAgent == "" {
			return fmt.Errorf("%w: the user agent is empty", ErrInvalidOption)
		}

		cfg.userAgent = userAgent

		return nil
	}
}

// WithProxy sends HTTP requests through a proxy, in place of the one the environment sets.
//
// Parameters:
//   - proxy: the proxy URL, with an http, https, socks5, or socks5h scheme and a host.
//
// Returns:
//   - Option: the option, which fails for a URL that does not parse or has another scheme or no host.
func WithProxy(proxy string) Option {
	return func(cfg *settings) error {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return fmt.Errorf("%w: proxy: %w", ErrInvalidOption, err)
		}

		if !slices.Contains(proxySchemes, proxyURL.Scheme) || proxyURL.Host == "" {
			return fmt.Errorf(
				"%w: proxy %q needs a host and one of the schemes %v",
				ErrInvalidOption,
				proxyURL.Redacted(),
				proxySchemes,
			)
		}

		cfg.proxy = proxyURL

		return nil
	}
}

// WithBaseURL sets the speedtest API base URL, in place of [DefaultBaseURL].
//
// Parameters:
//   - baseURL: the base URL, which [ParseBaseURL] must accept.
//
// Returns:
//   - Option: the option, which fails with an error that also wraps [ErrInvalidBaseURL] for an unusable URL.
func WithBaseURL(baseURL string) Option {
	return func(cfg *settings) error {
		_, err := ParseBaseURL(baseURL)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidOption, err)
		}

		cfg.baseURL = baseURL

		return nil
	}
}

// WithSource sets where the client's connections leave from.
//
// The source is an IP address, a network interface name, or a hostname, optionally with a network prefix such as
// "tcp://". On Linux, an interface name binds each socket to the interface, which then supplies an address of the
// destination's family. Elsewhere it uses the address [ResolveInterface] picks. The source applies to HTTP, TCP,
// ICMP, and UDP connections, and to DNS queries with [WithDNSBindSource].
//
// Parameters:
//   - source: the source.
//
// Returns:
//   - Option: the option, which fails for an empty source or one that does not resolve.
func WithSource(source string) Option {
	return func(cfg *settings) error {
		if source == "" {
			return fmt.Errorf("%w: the source is empty", ErrInvalidOption)
		}

		binding, err := resolveSource(source)
		if err != nil {
			return fmt.Errorf("%w: source %q: %w", ErrInvalidOption, source, err)
		}

		cfg.source = binding
		cfg.sourceName = source

		return nil
	}
}

// WithDNSBindSource sends DNS queries the way the client's other connections leave, from the [WithSource] source
// and through the [WithDialerControl] and [WithDSCP] socket settings. It uses the pure Go resolver.
//
// Returns:
//   - Option: the option.
func WithDNSBindSource() Option {
	return func(cfg *settings) error {
		cfg.dnsBindSource = true

		return nil
	}
}

// WithDSCP marks each socket the client dials with a Differentiated Services code point.
//
// Parameters:
//   - dscp: the code point from 0 to 63, where 0 leaves sockets unmarked.
//
// Returns:
//   - Option: the option, which fails with an error that also wraps the [CheckDSCP] error for an unusable value.
func WithDSCP(dscp int) Option {
	return func(cfg *settings) error {
		err := CheckDSCP(dscp)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidOption, err)
		}

		cfg.dscp = dscp

		return nil
	}
}

// WithDialerControl adds a function that runs on each socket the client dials before it connects.
//
// It runs after any binding to a source interface and any DSCP marking. Each call adds another function, and they
// run in the order they were added. With [WithDNSBindSource], they also run on DNS sockets.
//
// Parameters:
//   - control: the function, called like [net.Dialer] Control.
//
// Returns:
//   - Option: the option, which fails for a nil function.
func WithDialerControl(control func(network, address string, conn syscall.RawConn) error) Option {
	return func(cfg *settings) error {
		if control == nil {
			return fmt.Errorf("%w: the dialer control is nil", ErrInvalidOption)
		}

		cfg.controls = append(cfg.controls, control)

		return nil
	}
}

// WithPingMode sets how the client measures latency. The default is [HTTP].
//
// Parameters:
//   - mode: [HTTP], [TCP], or [ICMP].
//
// Returns:
//   - Option: the option, which fails for any other value.
func WithPingMode(mode Proto) Option {
	return func(cfg *settings) error {
		switch mode {
		case HTTP, TCP, ICMP:
			cfg.pingMode = mode

			return nil
		default:
			return fmt.Errorf("%w: unknown ping mode %d", ErrInvalidOption, mode)
		}
	}
}

// WithProtocol sets the protocol of download and upload tests. The default is [HTTPTest].
//
// Parameters:
//   - mode: [HTTPTest] or [TCPTest].
//
// Returns:
//   - Option: the option, which fails for any other value.
func WithProtocol(mode TestMode) Option {
	return func(cfg *settings) error {
		switch mode {
		case HTTPTest, TCPTest:
			cfg.testMode = mode

			return nil
		default:
			return fmt.Errorf("%w: unknown protocol %d", ErrInvalidOption, mode)
		}
	}
}

// WithConnections sets how many connections each transfer test uses. The default is one per CPU for downloads and
// up to 8 for uploads.
//
// Parameters:
//   - connections: the number of connections, at least 1.
//
// Returns:
//   - Option: the option, which fails for a value below 1.
func WithConnections(connections int) Option {
	return func(cfg *settings) error {
		if connections < 1 {
			return fmt.Errorf(
				"%w: connections must be at least 1, not %d",
				ErrInvalidOption,
				connections,
			)
		}

		cfg.connections = connections

		return nil
	}
}

// WithTransferSizes sets the block sizes of download and upload tests.
//
// Over HTTP the download size is the side of the requested image, and over TCP the size in kB. The upload size is
// in kB. A size of 0 keeps its default: 1000 for downloads, and 1000 for single-server or 800 for multi-server
// uploads.
//
// Parameters:
//   - download: the download size, 0 or one that [CheckDownloadSize] accepts.
//   - upload: the upload size, 0 or one that [CheckUploadSize] accepts.
//
// Returns:
//   - Option: the option, which fails with an error that also wraps [ErrInvalidTransferSize] for a size outside
//     its table.
func WithTransferSizes(download, upload int) Option {
	return func(cfg *settings) error {
		err := errors.Join(CheckDownloadSize(download), CheckUploadSize(upload))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidOption, err)
		}

		cfg.downloadSize = download
		cfg.uploadSize = upload

		return nil
	}
}

// WithTestDuration sets how long each download and upload test runs.
//
// Parameters:
//   - duration: the duration, greater than 0.
//
// Returns:
//   - Option: the option, which fails for a duration that is not positive.
func WithTestDuration(duration time.Duration) Option {
	return func(cfg *settings) error {
		if duration <= 0 {
			return fmt.Errorf(
				"%w: the test duration must be positive, not %s",
				ErrInvalidOption,
				duration,
			)
		}

		cfg.testDuration = duration

		return nil
	}
}

// WithLocation ranks servers by their distance from a location, in place of the location speedtest.net reports.
//
// Parameters:
//   - location: the location, such as one from [GetLocation] or [ParseLocation]. It is copied.
//
// Returns:
//   - Option: the option, which fails for a nil location.
func WithLocation(location *Location) Option {
	return func(cfg *settings) error {
		if location == nil {
			return fmt.Errorf("%w: the location is nil", ErrInvalidOption)
		}

		copied := *location
		cfg.location = &copied

		return nil
	}
}

// WithSearch limits the server list to servers that match a keyword, such as a city or sponsor name.
//
// Parameters:
//   - keyword: the keyword.
//
// Returns:
//   - Option: the option, which fails for an empty keyword.
func WithSearch(keyword string) Option {
	return func(cfg *settings) error {
		if keyword == "" {
			return fmt.Errorf("%w: the search keyword is empty", ErrInvalidOption)
		}

		cfg.search = keyword

		return nil
	}
}

// WithLogger sends the client's debug messages to a logger. Without it, the client discards them.
//
// Parameters:
//   - logger: the logger.
//
// Returns:
//   - Option: the option, which fails for a nil logger.
func WithLogger(logger Logger) Option {
	return func(cfg *settings) error {
		if logger == nil {
			return fmt.Errorf("%w: the logger is nil", ErrInvalidOption)
		}

		cfg.logger = logger

		return nil
	}
}

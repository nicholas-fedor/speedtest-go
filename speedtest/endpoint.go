package speedtest

import (
	"errors"
	"fmt"
	"net/url"
)

// DefaultBaseURL is the speedtest.net API base URL, used when [UserConfig] sets no BaseURL.
const DefaultBaseURL = "https://www.speedtest.net"

// API endpoint paths, relative to the base URL.
const (
	// userConfigPath returns the caller's IP, ISP, and coordinates.
	userConfigPath = "speedtest-config.php"
	// serversPath returns the server list as JSON.
	serversPath = "api/js/servers"
	// serversAlternativePath returns the server list as XML, used when serversPath returns an empty body.
	serversAlternativePath = "speedtest-servers-static.php"
	// serversAdvancedPath returns a single server and the caller's location as XML.
	serversAdvancedPath = "api/ios-config.php"
)

// ErrInvalidBaseURL is returned when a base URL is not an absolute http or https URL.
var ErrInvalidBaseURL = errors.New("invalid base URL")

// ParseBaseURL parses and validates a speedtest API base URL.
//
// The URL must be absolute, use http or https, name a host, and carry no credentials, query, or fragment.
// A path is allowed, so an API mirror can live under a prefix such as https://mirror.example/speedtest.
//
// Parameters:
//   - raw: the base URL to parse.
//
// Returns:
//   - *url.URL: the parsed base URL.
//   - error: an error wrapping [ErrInvalidBaseURL] when raw is not a usable base URL.
func ParseBaseURL(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBaseURL, err)
	}

	// Checked first and reported redacted, so later messages that quote raw never expose a password.
	if base.User != nil {
		return nil, fmt.Errorf(
			"%w: %q must not include credentials",
			ErrInvalidBaseURL,
			base.Redacted(),
		)
	}

	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("%w: %q must use http or https", ErrInvalidBaseURL, raw)
	}

	if base.Host == "" {
		return nil, fmt.Errorf("%w: %q has no host", ErrInvalidBaseURL, raw)
	}

	if base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("%w: %q must not have a query or fragment", ErrInvalidBaseURL, raw)
	}

	return base, nil
}

// endpoint resolves an API path against the client's base URL.
//
// It falls back to [DefaultBaseURL] when the client has no configuration or no base URL, and keeps any path
// prefix of the base URL.
//
// Parameters:
//   - path: the endpoint path relative to the base URL.
//
// Returns:
//   - *url.URL: the endpoint URL.
//   - error: an error wrapping [ErrInvalidBaseURL] when the configured base URL is invalid.
func (s *Speedtest) endpoint(path string) (*url.URL, error) {
	raw := DefaultBaseURL
	if s.config != nil && s.config.BaseURL != "" {
		raw = s.config.BaseURL
	}

	base, err := ParseBaseURL(raw)
	if err != nil {
		return nil, err
	}

	return base.JoinPath(path), nil
}

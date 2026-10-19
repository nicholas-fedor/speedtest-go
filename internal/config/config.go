// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Config holds the application configuration.
type Config struct {
	// CustomURL is the URL of a server to test in place of the ones speedtest.net lists, or empty.
	CustomURL string
	// Location is the user's location as "lat,lon" coordinates, or empty.
	Location string
	// City is a predefined city label that sets the user's location, or empty.
	City string
	// Proxy is the HTTP, HTTPS, or SOCKS proxy URL for the requests, or empty.
	Proxy string
	// Source is the IP address or network interface name to send from, or empty.
	Source string
	// Search is a keyword that servers are fuzzy matched against, or empty.
	Search string
	// UserAgent is the User-Agent header of the requests, or empty for the library's default.
	UserAgent string
	// BaseURL is the speedtest.net API base URL, or empty for the library's default.
	BaseURL string
	// ServerIDs are the IDs of the servers to test, or empty to choose servers by latency.
	ServerIDs []int
	// CountryCodes are the upper-case two-letter country codes servers must be in, with AUTO for the user's own
	// country. Empty keeps every server.
	CountryCodes []string
	// Thread is the number of concurrent connections of a transfer test, or 0 for one per CPU.
	Thread int
	// PingMode is the protocol of the latency measurement.
	PingMode speedtest.Proto
	// TestMode is the protocol for download and upload tests.
	TestMode speedtest.TestMode
	// Unit is the unit of the rates in the output. See UnitSet.
	Unit speedtest.UnitType
	// DSCP is the Differentiated Services code point to mark connections with, or 0 to leave them unmarked.
	DSCP int
	// DownloadSize is the download block size from the library's size table, or 0 for the default.
	DownloadSize int
	// UploadSize is the upload block size in kB from the library's size table, or 0 for the default.
	UploadSize int
	// SavingMode tests with one connection and plain line output, which uses few resources but is less accurate.
	SavingMode bool
	// JSONOutput writes one JSON document for the run.
	JSONOutput bool
	// JSONLOutput writes one JSON line per server.
	JSONLOutput bool
	// UnixOutput writes one plain line per result.
	UnixOutput bool
	// DNSBindSource sends DNS lookups from Source too.
	DNSBindSource bool
	// Multi spreads each transfer test across the main server and the other servers found.
	Multi bool
	// NoDownload skips the download test.
	NoDownload bool
	// NoUpload skips the upload test.
	NoUpload bool
	// UnitSet reports that Unit holds a resolved unit. The zero value of [speedtest.UnitType] is decimal bits,
	// so a Config without it uses the default Mbps output.
	UnitSet bool
	// Debug writes the library's debug messages to stderr and shows plain lines.
	Debug bool
	// MaskISP masks the user's IP address, coordinates, and distances to servers in every output.
	MaskISP bool
}

// CountryAuto is the country filter entry that stands for the user's own country.
const CountryAuto = "AUTO"

// EnvPrefix is the prefix of the environment variables that supply settings. A setting's variable is the prefix,
// an underscore, and the setting name in upper case with hyphens as underscores, such as SPEEDTEST_GO_PING_MODE.
const EnvPrefix = "SPEEDTEST_GO"

// Errors for setting values.
var (
	// ErrInvalidSetting is wrapped by every error that rejects a setting value. The CLI exits with code 2 for it.
	ErrInvalidSetting = errors.New("invalid setting")
	// errNotBoolean rejects a switch that does not parse as true or false.
	errNotBoolean = errors.New("not true or false")
	// errNotWholeNumber rejects a count that does not parse as an integer.
	errNotWholeNumber = errors.New("not a whole number")
	// errNegative rejects a count below zero.
	errNegative = errors.New("negative")
	// errUnknownSource rejects a source that is neither an IP address nor a network interface.
	errUnknownSource = errors.New("neither an IP address nor a network interface")
	// errInvalidCountryCode rejects a country filter entry that is neither a two-letter code nor auto.
	errInvalidCountryCode = errors.New("not a two-letter country code or auto")
	// errFilterWithServers rejects a country filter alongside servers that are already chosen.
	errFilterWithServers = errors.New(
		"--filter-cc cannot be used with --server or --custom-url, which choose the servers",
	)
	// errUnknownCity rejects a city that is not one of the predefined labels.
	errUnknownCity = errors.New("not a city label, run the cities command to list them")
	// errCityWithLocation rejects a city and coordinates together, which set the location twice.
	errCityWithLocation = errors.New("--city and --location both set the location, use one")
	// errLocationWithServers rejects location settings alongside servers that are already chosen.
	errLocationWithServers = errors.New(
		"--city, --location, and --search cannot be used with --server or --custom-url, which choose the servers",
	)
	// errProxyWithTCP rejects a proxy with TCP tests, whose connections cannot go through an HTTP proxy.
	errProxyWithTCP = errors.New(
		"--proxy cannot be used with --protocol tcp, whose connections bypass HTTP proxies",
	)
)

// Setup applies global side effects based on the configuration and returns
// the resulting Config so callers receive any defaults applied here.
//
// Without a resolved unit, rates are shown in Mbps.
//
// Parameters:
//   - cfg: the validated settings.
//
// Returns:
//   - Config: cfg with its defaults applied.
func Setup(cfg Config) Config {
	if !cfg.UnitSet {
		cfg.Unit = speedtest.UnitTypeDefaultMbps
		cfg.UnitSet = true
	}

	if cfg.SavingMode && !cfg.JSONOutput && !cfg.JSONLOutput && !cfg.UnixOutput {
		cfg.UnixOutput = true
	}

	if cfg.JSONOutput || cfg.JSONLOutput || cfg.UnixOutput {
		log.SetOutput(io.Discard)
		log.SetFlags(0)
	}

	return cfg
}

// parseServerIDs parses the IDs of the servers to test.
//
// Parameters:
//   - entries: the IDs as text.
//
// Returns:
//   - []int: the IDs, or nil when there are none.
//   - error: an error wrapping errNotWholeNumber naming the first entry that is not an integer.
func parseServerIDs(entries []string) ([]int, error) {
	var ids []int

	for _, entry := range entries {
		id, err := strconv.Atoi(strings.TrimSpace(entry))
		if err != nil {
			return nil, fmt.Errorf("%q is %w", entry, errNotWholeNumber)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// checkBaseURL rejects a base URL the client could not use.
//
// Parameters:
//   - raw: the setting value. Empty means the default.
//
// Returns:
//   - error: the error wrapping [speedtest.ErrInvalidBaseURL], or nil.
func checkBaseURL(raw string) error {
	if raw == "" {
		return nil
	}

	_, err := speedtest.ParseBaseURL(raw)
	if err != nil {
		return fmt.Errorf("cannot use this base URL: %w", err)
	}

	return nil
}

// checkCombinations rejects settings that are valid alone but not together.
//
// Parameters:
//   - cfg: the settings to check.
//
// Returns:
//   - []error: one error wrapping [ErrInvalidSetting] per problem, or none.
func checkCombinations(cfg Config) []error {
	var errs []error

	err := checkOutputFormats(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	errs = append(errs, checkLocation(cfg)...)

	if len(cfg.CountryCodes) > 0 && (len(cfg.ServerIDs) > 0 || cfg.CustomURL != "") {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidSetting, errFilterWithServers))
	}

	if cfg.TestMode == speedtest.TCPTest && cfg.Proxy != "" {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidSetting, errProxyWithTCP))
	}

	return errs
}

// parseThread parses the connection count.
//
// Parameters:
//   - raw: the setting value. Empty means zero, which lets the library pick.
//
// Returns:
//   - int: the connection count.
//   - error: the parse error, or an error for a negative count.
func parseThread(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	thread, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is %w", raw, errNotWholeNumber)
	}

	if thread < 0 {
		return 0, fmt.Errorf(
			"%d is %w: use 0 for the default or a positive count",
			thread,
			errNegative,
		)
	}

	return thread, nil
}

// parseDSCP parses the Differentiated Services code point.
//
// Parameters:
//   - raw: the setting value. Empty means 0, which leaves connections unmarked.
//
// Returns:
//   - int: the code point.
//   - error: the parse error, or an error wrapping [speedtest.ErrInvalidDSCP] or [speedtest.ErrDSCPUnsupported].
func parseDSCP(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	dscp, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is %w", raw, errNotWholeNumber)
	}

	err = speedtest.CheckDSCP(dscp)
	if err != nil {
		return 0, fmt.Errorf("cannot mark connections: %w", err)
	}

	return dscp, nil
}

// checkCity rejects a city that is not one of the predefined labels.
//
// Parameters:
//   - city: the setting value. Empty means no city.
//
// Returns:
//   - error: an error wrapping errUnknownCity, or nil.
func checkCity(city string) error {
	if city == "" {
		return nil
	}

	_, err := speedtest.GetLocation(city)
	if err != nil {
		return fmt.Errorf("%q is %w", city, errUnknownCity)
	}

	return nil
}

// checkCoordinates rejects coordinates that do not parse.
//
// Parameters:
//   - location: the setting value, as lat,lon. Empty means no coordinates.
//
// Returns:
//   - error: the parse error, or nil.
func checkCoordinates(location string) error {
	if location == "" {
		return nil
	}

	_, _, err := speedtest.ParseCoordinates(location)
	if err != nil {
		return fmt.Errorf("cannot use these coordinates: %w", err)
	}

	return nil
}

// checkLocation rejects location settings that conflict with each other or with the chosen servers.
//
// Parameters:
//   - cfg: the settings to check.
//
// Returns:
//   - []error: one error wrapping [ErrInvalidSetting] per problem, or none.
func checkLocation(cfg Config) []error {
	var errs []error

	if cfg.City != "" && cfg.Location != "" {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidSetting, errCityWithLocation))
	}

	choosesLocation := cfg.City != "" || cfg.Location != "" || cfg.Search != ""
	if choosesLocation && (len(cfg.ServerIDs) > 0 || cfg.CustomURL != "") {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidSetting, errLocationWithServers))
	}

	return errs
}

// parseCountryCodes normalizes the country filter.
//
// Entries may also hold comma-separated codes, as an environment variable does. Codes are upper-cased, blanks and
// repeats are dropped, and the order is kept.
//
// Parameters:
//   - entries: the setting values.
//
// Returns:
//   - []string: the upper-case codes, with AUTO for the user's own country, or nil when there are none.
//   - error: an error wrapping errInvalidCountryCode naming the first entry that is not a code or auto.
func parseCountryCodes(entries []string) ([]string, error) {
	var codes []string

	for _, entry := range entries {
		for code := range strings.SplitSeq(entry, ",") {
			code = strings.ToUpper(strings.TrimSpace(code))
			if code == "" || slices.Contains(codes, code) {
				continue
			}

			if code != CountryAuto && !isCountryCode(code) {
				return nil, fmt.Errorf("%q is %w", code, errInvalidCountryCode)
			}

			codes = append(codes, code)
		}
	}

	return codes, nil
}

// isCountryCode reports whether code is two ASCII letters.
//
// Parameters:
//   - code: the upper-case entry to check.
//
// Returns:
//   - bool: true for two upper-case ASCII letters.
func isCountryCode(code string) bool {
	return len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z'
}

// parseTransferSize parses a download or upload block size.
//
// Parameters:
//   - raw: the setting value. Empty means 0, the default size.
//   - check: the library's check for the direction, [speedtest.CheckDownloadSize] or [speedtest.CheckUploadSize].
//
// Returns:
//   - int: the size.
//   - error: the parse error, or the check's error wrapping [speedtest.ErrInvalidTransferSize].
func parseTransferSize(raw string, check func(int) error) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	size, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is %w", raw, errNotWholeNumber)
	}

	err = check(size)
	if err != nil {
		return 0, fmt.Errorf("cannot use this size: %w", err)
	}

	return size, nil
}

// checkSource rejects a source that the speedtest client could not bind to.
//
// Parameters:
//   - source: the setting value, an IP address or a network interface name, optionally with a network prefix such
//     as "tcp://". Empty means no source.
//
// Returns:
//   - error: an error wrapping errUnknownSource for any other value, or wrapping
//     [speedtest.ErrNoInterfaceAddress] for an interface without a usable address.
func checkSource(source string) error {
	if source == "" {
		return nil
	}

	_, address, found := strings.Cut(source, "://")
	if !found {
		address = source
	}

	if net.ParseIP(address) != nil {
		return nil
	}

	_, err := speedtest.ResolveInterface(address)
	if errors.Is(err, speedtest.ErrInterfaceNotFound) {
		return fmt.Errorf("%q is %w", source, errUnknownSource)
	}

	if err != nil {
		return fmt.Errorf("%q cannot be the source: %w", source, err)
	}

	return nil
}

// checkOutputFormats rejects more than one of the json, jsonl, and unix output formats.
//
// Parameters:
//   - cfg: the settings to check.
//
// Returns:
//   - error: an error wrapping [ErrInvalidSetting] that names the chosen formats.
func checkOutputFormats(cfg Config) error {
	var chosen []string

	for _, format := range []struct {
		name string
		set  bool
	}{
		{name: "--json", set: cfg.JSONOutput},
		{name: "--jsonl", set: cfg.JSONLOutput},
		{name: "--unix", set: cfg.UnixOutput},
	} {
		if format.set {
			chosen = append(chosen, format.name)
		}
	}

	if len(chosen) > 1 {
		return fmt.Errorf("%w: %s cannot be combined, choose one output format",
			ErrInvalidSetting, strings.Join(chosen, ", "))
	}

	return nil
}

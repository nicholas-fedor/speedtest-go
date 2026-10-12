// Package parser provides functions for parsing configuration strings like units and protocols.
package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Parse errors for setting values.
var (
	// ErrUnknownUnit is returned for a unit name that is not one of the supported units.
	ErrUnknownUnit = errors.New("unknown unit")
	// ErrUnknownPingMode is returned for a ping mode that is not http, tcp, or icmp.
	ErrUnknownPingMode = errors.New("unknown ping mode")
	// ErrUnknownProtocol is returned for a test protocol that is not http or tcp.
	ErrUnknownProtocol = errors.New("unknown protocol")
)

// ParseUnit parses a unit name, case-insensitively.
//
// Parameters:
//   - str: the unit name. Empty means the default Mbps output.
//
// Returns:
//   - speedtest.UnitType: the unit.
//   - error: an error wrapping [ErrUnknownUnit] that lists the accepted names.
func ParseUnit(str string) (speedtest.UnitType, error) {
	switch strings.ToLower(str) {
	case "":
		return speedtest.UnitTypeDefaultMbps, nil
	case "decimal-bits":
		return speedtest.UnitTypeDecimalBits, nil
	case "decimal-bytes":
		return speedtest.UnitTypeDecimalBytes, nil
	case "binary-bits":
		return speedtest.UnitTypeBinaryBits, nil
	case "binary-bytes":
		return speedtest.UnitTypeBinaryBytes, nil
	default:
		return speedtest.UnitTypeDefaultMbps, fmt.Errorf(
			"%w %q: use decimal-bits, decimal-bytes, binary-bits, or binary-bytes",
			ErrUnknownUnit,
			str,
		)
	}
}

// ParseProto parses a ping mode, case-insensitively.
//
// Parameters:
//   - str: the ping mode. Empty means HTTP.
//
// Returns:
//   - speedtest.Proto: the ping protocol.
//   - error: an error wrapping [ErrUnknownPingMode] that lists the accepted modes.
func ParseProto(str string) (speedtest.Proto, error) {
	switch strings.ToLower(str) {
	case "", "http":
		return speedtest.HTTP, nil
	case "tcp":
		return speedtest.TCP, nil
	case "icmp":
		return speedtest.ICMP, nil
	default:
		return speedtest.HTTP, fmt.Errorf("%w %q: use http, tcp, or icmp", ErrUnknownPingMode, str)
	}
}

// ParseTestMode parses the download and upload protocol, case-insensitively.
//
// Parameters:
//   - str: the protocol. Empty means HTTP.
//
// Returns:
//   - speedtest.TestMode: the test mode.
//   - error: an error wrapping [ErrUnknownProtocol] that lists the accepted protocols.
func ParseTestMode(str string) (speedtest.TestMode, error) {
	switch strings.ToLower(str) {
	case "", "http":
		return speedtest.HTTPTest, nil
	case "tcp":
		return speedtest.TCPTest, nil
	default:
		return speedtest.HTTPTest, fmt.Errorf("%w %q: use http or tcp", ErrUnknownProtocol, str)
	}
}

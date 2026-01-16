// Package parser provides functions for parsing configuration strings like units and protocols.
package parser

import (
	"strings"

	"github.com/showwin/speedtest-go/speedtest"
)

// ParseUnit parses the unit string to a UnitType.
func ParseUnit(str string) speedtest.UnitType {
	str = strings.ToLower(str)
	switch str {
	case "decimal-bits":
		return speedtest.UnitTypeDecimalBits
	case "decimal-bytes":
		return speedtest.UnitTypeDecimalBytes
	case "binary-bits":
		return speedtest.UnitTypeBinaryBits
	case "binary-bytes":
		return speedtest.UnitTypeBinaryBytes
	default:
		return speedtest.UnitTypeDefaultMbps
	}
}

// ParseProto parses the protocol string to a Proto.
func ParseProto(str string) speedtest.Proto {
	str = strings.ToLower(str)
	switch str {
	case "icmp":
		return speedtest.ICMP
	case "tcp":
		return speedtest.TCP
	default:
		return speedtest.HTTP
	}
}

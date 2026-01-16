package speedtest

import (
	"strconv"
)

// UnitType represents the type of unit for byte rate formatting.
type UnitType int

// IEC and SI.
const (
	UnitTypeDecimalBits  = UnitType(iota) // auto scaled
	UnitTypeDecimalBytes                  // auto scaled
	UnitTypeBinaryBits                    // auto scaled
	UnitTypeBinaryBytes                   // auto scaled
	UnitTypeDefaultMbps                   // fixed
)

var (
	// DecimalBitsUnits contains unit suffixes for decimal bit rates.
	DecimalBitsUnits = []string{"bps", "Kbps", "Mbps", "Gbps"}
	// DecimalBytesUnits contains unit suffixes for decimal byte rates.
	DecimalBytesUnits = []string{"B/s", "KB/s", "MB/s", "GB/s"}
	// BinaryBitsUnits contains unit suffixes for binary bit rates.
	BinaryBitsUnits = []string{"Kibps", "KiMbps", "KiGbps"}
	// BinaryBytesUnits contains unit suffixes for binary byte rates.
	BinaryBytesUnits = []string{"KiB/s", "MiB/s", "GiB/s"}
)

var unitMaps = map[UnitType][]string{
	UnitTypeDecimalBits:  DecimalBitsUnits,
	UnitTypeDecimalBytes: DecimalBytesUnits,
	UnitTypeBinaryBits:   BinaryBitsUnits,
	UnitTypeBinaryBytes:  BinaryBytesUnits,
}

const (
	// B represents 1 byte in decimal units.
	B = 1.0
	// Kilobyte represents 1000 bytes in decimal units.
	Kilobyte = 1000 * B
	// Megabyte represents 1000000 bytes in decimal units.
	Megabyte = 1000 * Kilobyte
	// Gigabyte represents 1000000000 bytes in decimal units.
	Gigabyte = 1000 * Megabyte

	// IB represents 1 byte in binary units.
	IB = 1
	// KiB represents 1024 bytes in binary units.
	KiB = 1024 * IB
	// MiB represents 1048576 bytes in binary units.
	MiB = 1024 * KiB
	// GiB represents 1073741824 bytes in binary units.
	GiB = 1024 * MiB
)

// ByteRate represents a byte rate value with formatting capabilities.
type ByteRate float64

var globalByteRateUnit UnitType

func (r ByteRate) String() string {
	if r == 0 {
		return "0.00 Mbps"
	}

	if r == -1 {
		return "N/A"
	}

	if globalByteRateUnit != UnitTypeDefaultMbps {
		return r.Byte(globalByteRateUnit)
	}

	return strconv.FormatFloat(float64(r/125000.0), 'f', 2, 64) + " Mbps"
}

// SetUnit Set global output units.
func SetUnit(unit UnitType) {
	globalByteRateUnit = unit
}

// Mbps returns the byte rate in megabits per second.
func (r ByteRate) Mbps() float64 {
	return float64(r) / 125000.0
}

// Gbps returns the byte rate in gigabits per second.
func (r ByteRate) Gbps() float64 {
	return float64(r) / 125000000.0
}

// Byte Specifies the format output byte rate.
func (r ByteRate) Byte(formatType UnitType) string {
	if r == 0 {
		return "0.00 Mbps"
	}

	if r == -1 {
		return "N/A"
	}

	return format(float64(r), formatType)
}

func format(byteRate float64, unitType UnitType) string {
	val := byteRate
	if unitType%2 == 0 {
		val *= 8
	}

	if unitType < UnitTypeBinaryBits {
		switch {
		case byteRate >= Gigabyte:
			return strconv.FormatFloat(val/Gigabyte, 'f', 2, 64) + " " + unitMaps[unitType][3]
		case byteRate >= Megabyte:
			return strconv.FormatFloat(val/Megabyte, 'f', 2, 64) + " " + unitMaps[unitType][2]
		case byteRate >= Kilobyte:
			return strconv.FormatFloat(val/Kilobyte, 'f', 2, 64) + " " + unitMaps[unitType][1]
		default:
			return strconv.FormatFloat(val/B, 'f', 2, 64) + " " + unitMaps[unitType][0]
		}
	}

	switch {
	case byteRate >= GiB:
		return strconv.FormatFloat(val/GiB, 'f', 2, 64) + " " + unitMaps[unitType][2]
	case byteRate >= MiB:
		return strconv.FormatFloat(val/MiB, 'f', 2, 64) + " " + unitMaps[unitType][1]
	default:
		return strconv.FormatFloat(val/KiB, 'f', 2, 64) + " " + unitMaps[unitType][0]
	}
}

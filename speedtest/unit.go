// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

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

// rateScale is a unit prefix that a scaled rate can be shown in.
type rateScale struct {
	// label is the unit, such as "Mbps".
	label string
	// size is the number of bits or bytes per second in one unit.
	size float64
}

// Prefixed units of each scaled unit type, from the smallest to the largest.
var (
	// decimalBitScales are the SI bit rate units.
	decimalBitScales = []rateScale{
		{"bps", B},
		{"Kbps", Kilobyte},
		{"Mbps", Megabyte},
		{"Gbps", Gigabyte},
	}
	// decimalByteScales are the SI byte rate units.
	decimalByteScales = []rateScale{
		{"B/s", B},
		{"KB/s", Kilobyte},
		{"MB/s", Megabyte},
		{"GB/s", Gigabyte},
	}
	// binaryBitScales are the IEC bit rate units.
	binaryBitScales = []rateScale{{"Kibps", KiB}, {"Mibps", MiB}, {"Gibps", GiB}}
	// binaryByteScales are the IEC byte rate units.
	binaryByteScales = []rateScale{{"KiB/s", KiB}, {"MiB/s", MiB}, {"GiB/s", GiB}}
)

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

	// StrZeroMbps is the string representation for zero byte rate.
	StrZeroMbps = "0.00 Mbps"
	// StrNotAvailable is the string representation for unavailable values.
	StrNotAvailable = "N/A"

	// bitsPerByte converts byte rates to bit rates.
	bitsPerByte = 8
)

// ByteRate represents a byte rate value with formatting capabilities.
type ByteRate float64

// String formats the rate in megabits per second, or as [StrNotAvailable] for -1.
//
// Returns:
//   - string: the formatted rate, such as "723.10 Mbps".
func (r ByteRate) String() string {
	return r.Format(UnitTypeDefaultMbps)
}

// Format formats the rate in a unit.
//
// The scaled unit types pick the largest prefix that keeps the value at 1 or more, so 1.5 Gbps is never shown as
// 1500 Mbps.
//
// Parameters:
//   - unit: the unit type. An unknown one formats in megabits per second.
//
// Returns:
//   - string: the formatted rate with two decimals and its unit, or [StrNotAvailable] for -1.
func (r ByteRate) Format(unit UnitType) string {
	if r == -1 {
		return StrNotAvailable
	}

	switch unit {
	case UnitTypeDecimalBits:
		return scaledRate(float64(r)*bitsPerByte, decimalBitScales)
	case UnitTypeDecimalBytes:
		return scaledRate(float64(r), decimalByteScales)
	case UnitTypeBinaryBits:
		return scaledRate(float64(r)*bitsPerByte, binaryBitScales)
	case UnitTypeBinaryBytes:
		return scaledRate(float64(r), binaryByteScales)
	case UnitTypeDefaultMbps:
		return strconv.FormatFloat(r.Mbps(), 'f', 2, 64) + " Mbps"
	default:
		return r.Format(UnitTypeDefaultMbps)
	}
}

// Mbps returns the byte rate in megabits per second.
func (r ByteRate) Mbps() float64 {
	return float64(r) / 125000.0
}

// Gbps returns the byte rate in gigabits per second.
func (r ByteRate) Gbps() float64 {
	return float64(r) / 125000000.0
}

// scaledRate formats a rate in the largest unit it fills at least once.
//
// Parameters:
//   - value: the rate in bits or bytes per second, matching scales.
//   - scales: the units, from the smallest to the largest.
//
// Returns:
//   - string: the formatted rate, in the smallest unit when it is below one of every unit.
func scaledRate(value float64, scales []rateScale) string {
	scale := scales[0]

	for _, candidate := range scales[1:] {
		if value >= candidate.size {
			scale = candidate
		}
	}

	return strconv.FormatFloat(value/scale.size, 'f', 2, 64) + " " + scale.label
}

package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestByteRate_String checks that String formats in megabits per second, and -1 as not available.
func TestByteRate_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0.00 Mbps", ByteRate(0).String())
	assert.Equal(t, "N/A", ByteRate(-1).String())
	assert.Equal(t, "1.00 Mbps", ByteRate(125000).String())
	assert.Equal(t, "2500.00 Mbps", ByteRate(312500000).String())
}

func TestByteRate_Mbps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    ByteRate
		want float64
	}{
		{
			name: "zero byte rate",
			r:    0,
			want: 0,
		},
		{
			name: "1 Mbps rate",
			r:    125000, // 125000 bytes/second = 1 Mbps
			want: 1.0,
		},
		{
			name: "10 Mbps rate",
			r:    1250000,
			want: 10.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.r.Mbps()
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestByteRate_Gbps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    ByteRate
		want float64
	}{
		{
			name: "zero byte rate",
			r:    0,
			want: 0,
		},
		{
			name: "1 Gbps rate",
			r:    125000000, // 125000000 bytes/second = 1 Gbps
			want: 1.0,
		},
		{
			name: "0.5 Gbps rate",
			r:    62500000,
			want: 0.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.r.Gbps()
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

// TestByteRate_Format checks every unit type at zero, not available, and rates that need each prefix.
func TestByteRate_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		rate ByteRate
		unit UnitType
	}{
		{name: "Mbps zero", rate: 0, unit: UnitTypeDefaultMbps, want: "0.00 Mbps"},
		{name: "Mbps not available", rate: -1, unit: UnitTypeDefaultMbps, want: "N/A"},
		{
			name: "Mbps fixed above a gigabit",
			rate: 312500000,
			unit: UnitTypeDefaultMbps,
			want: "2500.00 Mbps",
		},
		{name: "unknown unit uses Mbps", rate: 125000, unit: UnitType(99), want: "1.00 Mbps"},
		{name: "decimal bits zero", rate: 0, unit: UnitTypeDecimalBits, want: "0.00 bps"},
		{name: "decimal bits not available", rate: -1, unit: UnitTypeDecimalBits, want: "N/A"},
		{name: "decimal bits bps", rate: 100, unit: UnitTypeDecimalBits, want: "800.00 bps"},
		{name: "decimal bits Kbps", rate: 1500, unit: UnitTypeDecimalBits, want: "12.00 Kbps"},
		{name: "decimal bits Mbps", rate: 125000, unit: UnitTypeDecimalBits, want: "1.00 Mbps"},
		{name: "decimal bits Gbps", rate: 312500000, unit: UnitTypeDecimalBits, want: "2.50 Gbps"},
		{name: "decimal bytes zero", rate: 0, unit: UnitTypeDecimalBytes, want: "0.00 B/s"},
		{name: "decimal bytes B/s", rate: 999, unit: UnitTypeDecimalBytes, want: "999.00 B/s"},
		{name: "decimal bytes KB/s", rate: 1500, unit: UnitTypeDecimalBytes, want: "1.50 KB/s"},
		{name: "decimal bytes MB/s", rate: 1000000, unit: UnitTypeDecimalBytes, want: "1.00 MB/s"},
		{
			name: "decimal bytes GB/s",
			rate: 2000000000,
			unit: UnitTypeDecimalBytes,
			want: "2.00 GB/s",
		},
		{name: "binary bits zero", rate: 0, unit: UnitTypeBinaryBits, want: "0.00 Kibps"},
		{name: "binary bits not available", rate: -1, unit: UnitTypeBinaryBits, want: "N/A"},
		{name: "binary bits Kibps", rate: 1024, unit: UnitTypeBinaryBits, want: "8.00 Kibps"},
		{name: "binary bits Mibps", rate: 131072, unit: UnitTypeBinaryBits, want: "1.00 Mibps"},
		{name: "binary bits Gibps", rate: 134217728, unit: UnitTypeBinaryBits, want: "1.00 Gibps"},
		{name: "binary bytes zero", rate: 0, unit: UnitTypeBinaryBytes, want: "0.00 KiB/s"},
		{name: "binary bytes KiB/s", rate: 1536, unit: UnitTypeBinaryBytes, want: "1.50 KiB/s"},
		{name: "binary bytes MiB/s", rate: 1048576, unit: UnitTypeBinaryBytes, want: "1.00 MiB/s"},
		{
			name: "binary bytes GiB/s",
			rate: 1073741824,
			unit: UnitTypeBinaryBytes,
			want: "1.00 GiB/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.rate.Format(tt.unit))
		})
	}
}

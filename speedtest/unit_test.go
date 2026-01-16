package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteRate_String(t *testing.T) {
	tests := []struct {
		name string
		r    ByteRate
		want string
	}{
		{
			name: "zero byte rate",
			r:    0,
			want: "0.00 Mbps",
		},
		{
			name: "negative one byte rate",
			r:    -1,
			want: "N/A",
		},
		{
			name: "normal byte rate",
			r:    125000,         // 1 Mbps
			want: "1000.00 Kbps", // Uses global unit (UnitTypeDecimalBits by default)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.r.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetUnit(t *testing.T) {
	type args struct {
		unit UnitType
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "set decimal bits unit",
			args: args{unit: UnitTypeDecimalBits},
		},
		{
			name: "set binary bytes unit",
			args: args{unit: UnitTypeBinaryBytes},
		},
		{
			name: "set default Mbps unit",
			args: args{unit: UnitTypeDefaultMbps},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			SetUnit(tt.args.unit)
			// Test passes if no panic
		})
	}
}

func TestByteRate_Mbps(t *testing.T) {
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

func TestByteRate_Byte(t *testing.T) {
	type args struct {
		formatType UnitType
	}

	tests := []struct {
		name string
		r    ByteRate
		args args
		want string
	}{
		{
			name: "zero byte rate",
			r:    0,
			args: args{formatType: UnitTypeDecimalBytes},
			want: "0.00 Mbps",
		},
		{
			name: "negative one byte rate",
			r:    -1,
			args: args{formatType: UnitTypeDecimalBytes},
			want: "N/A",
		},
		{
			name: "decimal bytes format",
			r:    1000000, // 1 MB/s
			args: args{formatType: UnitTypeDecimalBytes},
			want: "1.00 MB/s",
		},
		{
			name: "binary bytes format",
			r:    1048576, // 1 MiB/s
			args: args{formatType: UnitTypeBinaryBytes},
			want: "1.00 MiB/s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.r.Byte(tt.args.formatType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_format(t *testing.T) {
	type args struct {
		byteRate float64
		i        UnitType
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "decimal bytes - KB",
			args: args{byteRate: 1500, i: UnitTypeDecimalBytes},
			want: "1.50 KB/s",
		},
		{
			name: "decimal bits - Kbps",
			args: args{byteRate: 125000, i: UnitTypeDecimalBits},
			want: "1000.00 Kbps",
		},
		{
			name: "binary bytes - KiB",
			args: args{byteRate: 1536, i: UnitTypeBinaryBytes},
			want: "1.50 KiB/s",
		},
		{
			name: "binary bits - Kibps",
			args: args{byteRate: 1024, i: UnitTypeBinaryBits},
			want: "8.00 Kibps",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := format(tt.args.byteRate, tt.args.i)
			assert.Equal(t, tt.want, got)
		})
	}
}

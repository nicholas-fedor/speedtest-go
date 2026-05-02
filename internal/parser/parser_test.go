package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

const (
	strDecimalBits  = "decimal-bits"
	strDecimalBytes = "decimal-bytes"
	strBinaryBits   = "binary-bits"
	strBinaryBytes  = "binary-bytes"
	strICMP         = "icmp"
	strTCP          = "tcp"
)

func TestParseUnit(t *testing.T) {
	t.Parallel()

	type args struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want speedtest.UnitType
	}{
		{
			name: strDecimalBits,
			args: args{str: strDecimalBits},
			want: speedtest.UnitTypeDecimalBits,
		},
		{
			name: strDecimalBytes,
			args: args{str: strDecimalBytes},
			want: speedtest.UnitTypeDecimalBytes,
		},
		{
			name: strBinaryBits,
			args: args{str: strBinaryBits},
			want: speedtest.UnitTypeBinaryBits,
		},
		{
			name: strBinaryBytes,
			args: args{str: strBinaryBytes},
			want: speedtest.UnitTypeBinaryBytes,
		},
		{
			name: "default",
			args: args{str: "unknown"},
			want: speedtest.UnitTypeDefaultMbps,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseUnit(tt.args.str)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseProto(t *testing.T) {
	t.Parallel()

	type args struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want speedtest.Proto
	}{
		{
			name: strICMP,
			args: args{str: strICMP},
			want: speedtest.ICMP,
		},
		{
			name: strTCP,
			args: args{str: strTCP},
			want: speedtest.TCP,
		},
		{
			name: "default http",
			args: args{str: "unknown"},
			want: speedtest.HTTP,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseProto(tt.args.str)
			assert.Equal(t, tt.want, got)
		})
	}
}

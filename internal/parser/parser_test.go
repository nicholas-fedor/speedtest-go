package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

func TestParseUnit(t *testing.T) {
	type args struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want speedtest.UnitType
	}{
		{
			name: "decimal-bits",
			args: args{str: "decimal-bits"},
			want: speedtest.UnitTypeDecimalBits,
		},
		{
			name: "decimal-bytes",
			args: args{str: "decimal-bytes"},
			want: speedtest.UnitTypeDecimalBytes,
		},
		{
			name: "binary-bits",
			args: args{str: "binary-bits"},
			want: speedtest.UnitTypeBinaryBits,
		},
		{
			name: "binary-bytes",
			args: args{str: "binary-bytes"},
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
	type args struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want speedtest.Proto
	}{
		{
			name: "icmp",
			args: args{str: "icmp"},
			want: speedtest.ICMP,
		},
		{
			name: "tcp",
			args: args{str: "tcp"},
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

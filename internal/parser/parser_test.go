// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// TestParseUnit checks every unit name, case-insensitivity, the empty default, and that unknown names are rejected.
func TestParseUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		str     string
		want    speedtest.UnitType
	}{
		{name: "empty", str: "", want: speedtest.UnitTypeDefaultMbps},
		{name: "decimal-bits", str: "decimal-bits", want: speedtest.UnitTypeDecimalBits},
		{name: "decimal-bytes", str: "decimal-bytes", want: speedtest.UnitTypeDecimalBytes},
		{name: "binary-bits", str: "binary-bits", want: speedtest.UnitTypeBinaryBits},
		{name: "binary-bytes", str: "binary-bytes", want: speedtest.UnitTypeBinaryBytes},
		{name: "upper case", str: "Binary-Bytes", want: speedtest.UnitTypeBinaryBytes},
		{
			name:    "unknown",
			str:     "unknown",
			want:    speedtest.UnitTypeDefaultMbps,
			wantErr: ErrUnknownUnit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseUnit(tt.str)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), tt.str)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseProto checks every ping mode, case-insensitivity, the empty default, and that unknown modes are rejected.
func TestParseProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		str     string
		want    speedtest.Proto
	}{
		{name: "empty", str: "", want: speedtest.HTTP},
		{name: "http", str: "http", want: speedtest.HTTP},
		{name: "tcp", str: "tcp", want: speedtest.TCP},
		{name: "icmp", str: "icmp", want: speedtest.ICMP},
		{name: "upper case", str: "ICMP", want: speedtest.ICMP},
		{name: "unknown", str: "udp", want: speedtest.HTTP, wantErr: ErrUnknownPingMode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseProto(tt.str)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), tt.str)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestParseTestMode checks every protocol, case-insensitivity, the empty default, and that unknown protocols are
// rejected.
func TestParseTestMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		str     string
		want    speedtest.TestMode
	}{
		{name: "empty", str: "", want: speedtest.HTTPTest},
		{name: "http", str: "http", want: speedtest.HTTPTest},
		{name: "tcp", str: "tcp", want: speedtest.TCPTest},
		{name: "upper case", str: "TCP", want: speedtest.TCPTest},
		{name: "unknown", str: "quic", want: speedtest.HTTPTest, wantErr: ErrUnknownProtocol},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseTestMode(tt.str)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), tt.str)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

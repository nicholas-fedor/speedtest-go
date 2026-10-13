// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package output

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// TestMaskUser checks that the IP address and coordinates are masked with their separators kept, that the ISP
// name and country are kept, and that the given user is left unchanged.
func TestMaskUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		user *speedtest.User
		want *speedtest.User
		name string
	}{
		{
			name: "IPv4",
			user: &speedtest.User{
				IP:      "203.0.113.7",
				Lat:     "35.6812",
				Lon:     "139.7671",
				Isp:     "Example ISP",
				Country: "JP",
			},
			want: &speedtest.User{
				IP:      "***.*.***.*",
				Lat:     "**.****",
				Lon:     "***.****",
				Isp:     "Example ISP",
				Country: "JP",
			},
		},
		{
			name: "IPv6 and negative coordinates",
			user: &speedtest.User{
				IP:      "2001:db8::1",
				Lat:     "-33.9",
				Lon:     "-70.6",
				Isp:     "Example ISP",
				Country: "CL",
			},
			want: &speedtest.User{
				IP:      "****:***::*",
				Lat:     "-**.*",
				Lon:     "-**.*",
				Isp:     "Example ISP",
				Country: "CL",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			original := *tt.user

			assert.Equal(t, tt.want, MaskUser(tt.user))
			assert.Equal(t, original, *tt.user, "the given user must not change")
		})
	}

	assert.Nil(t, MaskUser(nil))
}

// TestMaskedServerLine checks that the distance in a server's description is masked and the rest is kept, for each
// form the description takes.
func TestMaskedServerLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		server *speedtest.Server
		want   string
		name   string
	}{
		{
			name: "with country",
			server: &speedtest.Server{
				ID:       "6691",
				Distance: 9.03,
				Name:     "Shizuoka",
				Country:  "Japan",
				Sponsor:  "sudosan",
			},
			want: "[6691] *.**km Shizuoka (Japan) by sudosan",
		},
		{
			name: "without country",
			server: &speedtest.Server{
				ID:       "6691",
				Distance: 120.55,
				Name:     "Shizuoka",
				Sponsor:  "sudosan",
			},
			want: "[6691] ***.**km Shizuoka by sudosan",
		},
		{
			name:   "custom server",
			server: &speedtest.Server{ID: "Custom", Name: "speed.example", Sponsor: "?"},
			want:   "[Custom] speed.example",
		},
		{name: "nil", server: nil, want: "<nil server>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MaskedServerLine(tt.server)
			assert.Equal(t, tt.want, got)

			// Apart from the distance, the line must match the library's own description, so the two formats
			// cannot drift apart unnoticed.
			if tt.server != nil {
				distance := fmt.Sprintf("%.2f", tt.server.Distance)
				assert.Equal(
					t,
					tt.server.String(),
					strings.Replace(got, maskValue(distance)+"km", distance+"km", 1),
				)
			}
		})
	}
}

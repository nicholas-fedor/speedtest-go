// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package output

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// maskRune replaces each hidden character of a masked value.
const maskRune = '*'

// MaskUser returns a copy of the user with the IP address and coordinates masked, for --mask-isp.
//
// Letters and digits become asterisks, while separators such as "." and ":" stay, so the shape of the value remains
// readable. The ISP name and country are kept. The given user is not changed.
//
// Parameters:
//   - user: the user to mask, or nil.
//
// Returns:
//   - *speedtest.User: the masked copy, or nil for a nil user.
func MaskUser(user *speedtest.User) *speedtest.User {
	if user == nil {
		return nil
	}

	masked := *user
	masked.IP = maskValue(user.IP)
	masked.Lat = maskValue(user.Lat)
	masked.Lon = maskValue(user.Lon)

	return &masked
}

// maskValue replaces the letters and digits of a value with asterisks.
//
// Parameters:
//   - value: the value to mask.
//
// Returns:
//   - string: the masked value.
func maskValue(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return maskRune
		}

		return r
	}, value)
}

// MaskedServerLine describes a server for --mask-isp, with its distance from the user masked, since distances to
// servers at known coordinates reveal where the user is.
//
// It builds the line in the formats of [speedtest.Server.String] rather than editing that output, so a change to
// those formats can never leave the distance unmasked.
//
// Parameters:
//   - server: the server.
//
// Returns:
//   - string: the description, with the distance's digits masked.
func MaskedServerLine(server *speedtest.Server) string {
	// A nil or custom server's description carries no distance.
	if server == nil || server.Sponsor == "?" {
		return server.String()
	}

	distance := maskValue(fmt.Sprintf("%.2f", server.Distance)) + "km"

	if len(server.Country) == 0 {
		return fmt.Sprintf("[%4s] %s %s by %s", server.ID, distance, server.Name, server.Sponsor)
	}

	return fmt.Sprintf(
		"[%4s] %s %s (%s) by %s",
		server.ID,
		distance,
		server.Name,
		server.Country,
		server.Sponsor,
	)
}

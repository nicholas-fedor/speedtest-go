// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
)

// Errors for user information lookups.
var (
	// ErrInstanceNil is returned when the speedtest instance is nil.
	ErrInstanceNil = errors.New("speedtest instance is nil")
	// ErrFetchUserInfo is returned when fetching user information fails.
	ErrFetchUserInfo = errors.New("failed to fetch user information")
)

// User represents information determined about the caller by speedtest.net.
type User struct {
	// IP is the caller's public IP address.
	IP string `json:"ip" xml:"ip,attr"`
	// Lat is the caller's latitude in degrees, as text.
	Lat string `json:"lat" xml:"lat,attr"`
	// Lon is the caller's longitude in degrees, as text.
	Lon string `json:"lon" xml:"lon,attr"`
	// Isp is the name of the caller's internet service provider.
	Isp string `json:"isp" xml:"isp,attr"`
	// Country is the caller's ISO 3166-1 alpha-2 country code, as speedtest.net reports it.
	Country string `json:"country" xml:"country,attr"`
}

// Users for decode xml.
type Users struct {
	// Users are the client entries of the response.
	Users []User `xml:"client"`
}

// FetchUserInfo returns information about the caller determined by speedtest.net. The client does not keep it.
//
// Parameters:
//   - ctx: cancellation and deadline for the request.
//
// Returns:
//   - *User: the caller's IP address, ISP, coordinates, and country.
//   - error: [ErrInstanceNil] for a nil client, an error wrapping [ErrUnexpectedStatus] for a status outside 2xx,
//     [ErrFetchUserInfo] when the reply describes no caller, or the request or decoding error.
func (s *Client) FetchUserInfo(ctx context.Context) (*User, error) {
	if s == nil {
		return nil, ErrInstanceNil
	}

	configURL, err := s.endpoint(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build user info URL: %w", err)
	}

	cacheBust(configURL)

	s.log().Debug(ctx, "retrieving user info", logKeyURL, configURL.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	err = checkStatus(resp)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}

	// Decode xml
	decoder := xml.NewDecoder(resp.Body)

	var users Users

	err = decoder.Decode(&users)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	if len(users.Users) == 0 {
		return nil, ErrFetchUserInfo
	}

	user := users.Users[0]

	return &user, nil
}

// String representation of User.
//
// Returns:
//   - string: the IP address, ISP, and coordinates, such as "203.0.113.7 (Example ISP) [35.6812, 139.7671] ".
func (u *User) String() string {
	extInfo := ""

	return fmt.Sprintf("%s (%s) [%s, %s] %s", u.IP, u.Isp, u.Lat, u.Lon, extInfo)
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"encoding/xml"
)

// User is the caller identity the fake reports from the user info and server lookup endpoints.
type User struct {
	// IP is the caller's public address.
	IP string
	// Lat is the caller's latitude, as a decimal string.
	Lat string
	// Lon is the caller's longitude, as a decimal string.
	Lon string
	// ISP is the caller's internet service provider.
	ISP string
	// Country is the caller's ISO 3166-1 alpha-2 country code.
	Country string
}

// Server is a speedtest server the fake lists. Its URL and host always point back at the fake itself.
type Server struct {
	// ID is the server's speedtest.net identifier.
	ID string
	// Name is the server's city name.
	Name string
	// Country is the server's country name.
	Country string
	// CC is the server's ISO 3166-1 alpha-2 country code.
	CC string
	// Sponsor is the organization that hosts the server.
	Sponsor string
	// Lat is the server's latitude, as a decimal string.
	Lat string
	// Lon is the server's longitude, as a decimal string.
	Lon string
}

// wireServer is one server in the JSON server list.
//
//nolint:tagliatelle // Field names must match the speedtest.net server list format.
type wireServer struct {
	// URL is the server's upload URL.
	URL string `json:"url"`
	// Lat is the server's latitude.
	Lat string `json:"lat"`
	// Lon is the server's longitude.
	Lon string `json:"lon"`
	// Name is the server's city name.
	Name string `json:"name"`
	// Country is the server's country name.
	Country string `json:"country"`
	// CC is the server's country code.
	CC string `json:"cc"`
	// Sponsor is the organization that hosts the server.
	Sponsor string `json:"sponsor"`
	// ID is the server's identifier.
	ID string `json:"id"`
	// Host is the server's host and port.
	Host string `json:"host"`
}

// xmlServer is one server element in the XML server list and the server lookup.
type xmlServer struct {
	// URL is the server's upload URL.
	URL string `xml:"url,attr"`
	// Lat is the server's latitude.
	Lat string `xml:"lat,attr"`
	// Lon is the server's longitude.
	Lon string `xml:"lon,attr"`
	// Name is the server's city name.
	Name string `xml:"name,attr"`
	// Country is the server's country name.
	Country string `xml:"country,attr"`
	// CC is the server's country code.
	CC string `xml:"cc,attr"`
	// Sponsor is the organization that hosts the server.
	Sponsor string `xml:"sponsor,attr"`
	// ID is the server's identifier.
	ID string `xml:"id,attr"`
	// Host is the server's host and port.
	Host string `xml:"host,attr"`
}

// xmlClient is the client element that carries the caller's details.
type xmlClient struct {
	// IP is the caller's public address.
	IP string `xml:"ip,attr"`
	// Lat is the caller's latitude.
	Lat string `xml:"lat,attr"`
	// Lon is the caller's longitude.
	Lon string `xml:"lon,attr"`
	// ISP is the caller's internet service provider.
	ISP string `xml:"isp,attr"`
	// Country is the caller's country code.
	Country string `xml:"country,attr"`
}

// xmlSettings is the root element shared by the user info, XML server list, and server lookup responses.
type xmlSettings struct {
	// XMLName names the root element.
	XMLName xml.Name `xml:"settings"`
	// Client is the caller's details, omitted from the XML server list.
	Client *xmlClient `xml:"client,omitempty"`
	// Servers is the list of servers, omitted from the user info response.
	Servers []xmlServer `xml:"servers>server,omitempty"`
}

// DefaultUser returns the caller identity a new [API] reports.
//
// The address is from the TEST-NET-3 documentation range, so it can never belong to a real caller.
//
// Returns:
//   - User: a caller in Tokyo.
func DefaultUser() User {
	return User{
		IP:      "203.0.113.7",
		Lat:     "35.6812",
		Lon:     "139.7671",
		ISP:     "Example ISP",
		Country: "JP",
	}
}

// DefaultServers returns the servers a new [API] lists.
//
// Returns:
//   - []Server: a server in Osaka with ID 1002 and a server in Tokyo with ID 1001, in that order, so that sorting
//     by distance from [DefaultUser] reorders them.
func DefaultServers() []Server {
	return []Server{
		{
			ID:      "1002",
			Name:    "Osaka",
			Country: "Japan",
			CC:      "JP",
			Sponsor: "Example Networks",
			Lat:     "34.6937",
			Lon:     "135.5023",
		},
		{
			ID:      "1001",
			Name:    "Tokyo",
			Country: "Japan",
			CC:      "JP",
			Sponsor: "Example Networks",
			Lat:     "35.6895",
			Lon:     "139.6917",
		},
	}
}

// toXMLClient converts a user to its XML client element.
//
// Parameters:
//   - user: the caller identity.
//
// Returns:
//   - *xmlClient: the client element.
func toXMLClient(user User) *xmlClient {
	return &xmlClient{
		IP:      user.IP,
		Lat:     user.Lat,
		Lon:     user.Lon,
		ISP:     user.ISP,
		Country: user.Country,
	}
}

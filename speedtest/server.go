// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"cmp"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// payloadType is the format of a server list response.
type payloadType int

// Formats of the server list responses.
const (
	// typeJSONPayload is the JSON list.
	typeJSONPayload payloadType = iota
	// typeXMLPayload is the alternative XML list.
	typeXMLPayload
)

// Errors for server lookups.
var (
	// ErrServerNotFound is returned when no server is available or found.
	ErrServerNotFound = errors.New("no server available or found")

	// errSpeedtestClientNil rejects a call on a nil client.
	errSpeedtestClientNil = errors.New("speedtest client is nil")
	// errHostEmpty rejects a custom server without a URL.
	errHostEmpty = errors.New("host cannot be empty")
	// errPayloadDecode rejects a server list in a format the client cannot decode.
	errPayloadDecode = errors.New("response payload decoding not implemented")
)

// Server information.
type Server struct {
	// URL is the server's upload URL, such as "http://speed.example:8080/speedtest/upload.php".
	URL string `json:"url" xml:"url,attr"`
	// Lat is the server's latitude in degrees, as text.
	Lat string `json:"lat" xml:"lat,attr"`
	// Lon is the server's longitude in degrees, as text.
	Lon string `json:"lon" xml:"lon,attr"`
	// Name is the server's city.
	Name string `json:"name" xml:"name,attr"`
	// Country is the server's country name.
	Country string `json:"country" xml:"country,attr"`
	// CC is the server's two-letter country code.
	CC string `json:"cc" xml:"cc,attr"`
	// Sponsor is the organization that runs the server.
	Sponsor string `json:"sponsor" xml:"sponsor,attr"`
	// ID is the server's speedtest.net ID.
	ID string `json:"id" xml:"id,attr"`
	// Host is the server's host:port, which packet loss and TCP tests connect to.
	Host string `json:"host" xml:"host,attr"`
	// Distance is the distance in kilometres from the location set with [WithLocation], or else from the caller.
	Distance float64 `json:"distance" xml:"-"`
	// Latency is the round trip of the quick probe that [Client.FetchServers] sends to each listed server, or
	// [PingTimeout] when the probe failed. [Servers.Available] and [Servers.FindServer] rank servers by it.
	// [Client.Ping] measures latency properly.
	Latency time.Duration `json:"latency" xml:"-"`
}

// CustomServer given a URL string, return a new Server object, with as much
// filled in as we can.
//
// Parameters:
//   - host: the server's URL. Its path is replaced with the upload path.
//
// Returns:
//   - *Server: the server, with "?" for the details a URL does not give.
//   - error: an error for a nil client, an empty host, or a URL that does not parse.
func (s *Client) CustomServer(host string) (*Server, error) {
	if s == nil {
		return nil, errSpeedtestClientNil
	}

	if host == "" {
		return nil, errHostEmpty
	}

	parsedURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host URL: %w", err)
	}

	parsedURL.Path = "/speedtest/upload.php"
	parseHost := parsedURL.String()

	return &Server{
		ID:      "Custom",
		Lat:     "?",
		Lon:     "?",
		Country: "?",
		URL:     parseHost,
		Name:    parsedURL.Host,
		Host:    parsedURL.Host,
		Sponsor: "?",
	}, nil
}

// ServerList list of Server
// Users(Client) is also present in the serversAdvancedPath response.
type ServerList struct {
	// XMLName is the XML list's root element.
	XMLName xml.Name `json:"-" xml:"settings"`
	// Servers are the listed servers.
	Servers []*Server `json:"servers" xml:"servers>server"`
	// Users are the caller's details, with the coordinates distances are measured from.
	Users []User `json:"users" xml:"client"`
}

// Servers for sorting servers.
type Servers []*Server

// ByDistance for sorting servers.
type ByDistance struct {
	// Servers are the servers to sort.
	Servers
}

// Available returns servers that have valid latency measurements.
//
// Returns:
//   - *Servers: the servers whose probe succeeded, sorted by latency.
func (servers Servers) Available() *Servers {
	retServer := Servers{}

	for _, server := range servers {
		if server.Latency != PingTimeout {
			retServer = append(retServer, server)
		}
	}

	for i := range len(retServer) - 1 {
		for j := range len(retServer) - i - 1 {
			if retServer[j].Latency > retServer[j+1].Latency {
				retServer[j], retServer[j+1] = retServer[j+1], retServer[j]
			}
		}
	}

	return &retServer
}

// Len finds length of servers. For sorting servers.
//
// Returns:
//   - int: the number of servers.
func (servers Servers) Len() int {
	return len(servers)
}

// Swap swaps i-th and j-th. For sorting servers.
//
// Parameters:
//   - i: the index of one server.
//   - j: the index of the other.
func (servers Servers) Swap(i, j int) {
	servers[i], servers[j] = servers[j], servers[i]
}

// FilterCountry returns the servers in any of the given countries, in their original order.
//
// Parameters:
//   - codes: ISO 3166-1 alpha-2 country codes, such as "GB", matched without regard to case.
//
// Returns:
//   - Servers: the matching servers, empty when none match.
func (servers Servers) FilterCountry(codes ...string) Servers {
	filtered := Servers{}

	for _, server := range servers {
		if slices.ContainsFunc(
			codes,
			func(code string) bool { return strings.EqualFold(code, server.CC) },
		) {
			filtered = append(filtered, server)
		}
	}

	return filtered
}

// Hosts return hosts of servers.
//
// Returns:
//   - []string: each server's host:port in order, or nil when there are no servers.
func (servers Servers) Hosts() []string {
	if len(servers) == 0 {
		return nil
	}

	retServer := make([]string, 0, len(servers))
	for _, server := range servers {
		retServer = append(retServer, server.Host)
	}

	return retServer
}

// Less compares the distance. For sorting servers.
//
// Parameters:
//   - i: the index of one server.
//   - j: the index of the other.
//
// Returns:
//   - bool: true when server i is closer than server j.
func (b ByDistance) Less(i, j int) bool {
	return b.Servers[i].Distance < b.Servers[j].Distance
}

// FetchServerByID retrieves a server by its ID.
//
// Parameters:
//   - ctx: cancellation and deadline for the request.
//   - serverID: the server's ID.
//
// Returns:
//   - *Server: the server, with its distance from the location set with [WithLocation], or else from the caller.
//   - error: [ErrServerNotFound] when speedtest.net does not list the ID, an error wrapping [ErrUnexpectedStatus]
//     for a status outside 2xx, or the request or decoding error.
func (s *Client) FetchServerByID(ctx context.Context, serverID string) (*Server, error) {
	if s == nil {
		return nil, errSpeedtestClientNil
	}

	parsedURL, err := s.endpoint(serversAdvancedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build server lookup URL: %w", err)
	}

	query := parsedURL.Query()
	query.Set(strings.ToLower("serverID"), serverID)
	parsedURL.RawQuery = query.Encode()

	// The response also describes the caller, whose coordinates set the server's distance unless a location is set.
	cacheBust(parsedURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
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
		return nil, fmt.Errorf("server lookup request failed: %w", err)
	}

	var list ServerList

	decoder := xml.NewDecoder(resp.Body)

	err = decoder.Decode(&list)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	for serverIndex := range list.Servers {
		if list.Servers[serverIndex].ID == serverID {
			// The lookup lists servers without the host that packet loss and TCP tests connect to, so it comes from
			// the URL, with the port the server list would report.
			if list.Servers[serverIndex].Host == "" {
				host, hostErr := tcpHost(list.Servers[serverIndex])
				if hostErr == nil {
					list.Servers[serverIndex].Host = host
				}
			}

			// Distances are measured from the location set with WithLocation, as for the server list, and otherwise
			// from the caller.
			var originLat, originLon float64

			switch {
			case s.settings.location != nil:
				originLat, originLon = s.settings.location.Lat, s.settings.location.Lon
			case len(list.Users) > 0:
				originLat, _ = strconv.ParseFloat(list.Users[0].Lat, 64)
				originLon, _ = strconv.ParseFloat(list.Users[0].Lon, 64)
			default:
				return list.Servers[serverIndex], nil
			}

			sLat, _ := strconv.ParseFloat(list.Servers[serverIndex].Lat, 64)
			sLon, _ := strconv.ParseFloat(list.Servers[serverIndex].Lon, 64)
			list.Servers[serverIndex].Distance = distance(sLat, sLon, originLat, originLon)

			return list.Servers[serverIndex], nil
		}
	}

	return nil, ErrServerNotFound
}

// pingServers sends each server one quick probe with the client's ping mode and records its round trip as the
// server's latency, or [PingTimeout] when the probe fails.
//
// Parameters:
//   - ctx: ends the probes early.
//   - servers: the servers.
func (s *Client) pingServers(ctx context.Context, servers Servers) {
	var waitGroup sync.WaitGroup

	pCtx, cancelFunc := context.WithTimeout(ctx, time.Second*4)

	for _, server := range servers {
		waitGroup.Add(1)

		go func(serverPtr *Server) {
			var (
				latency []int64
				errPing error
			)

			latency, errPing = s.pingWithMode(
				pCtx,
				serverPtr,
				pingSettings{onLatency: nil, count: 1, interval: time.Millisecond},
			)

			if errPing != nil || len(latency) < 1 {
				serverPtr.Latency = PingTimeout
			} else {
				serverPtr.Latency = time.Duration(latency[0]) * time.Nanosecond
			}

			waitGroup.Done()
		}(server)
	}

	waitGroup.Wait()
	cancelFunc()
}

// buildServerListURL constructs the URL for fetching server list.
//
// Returns:
//   - *url.URL: the list's URL, with the search keyword and location as query parameters when set.
//   - error: the error from building the URL.
func (s *Client) buildServerListURL() (*url.URL, error) {
	parsedURL, err := s.endpoint(serversPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build server list URL: %w", err)
	}

	query := parsedURL.Query()
	if len(s.settings.search) > 0 {
		query.Set("search", s.settings.search)
	}

	if s.settings.location != nil {
		query.Set("lat", strconv.FormatFloat(s.settings.location.Lat, 'f', -1, 64))
		query.Set("lon", strconv.FormatFloat(s.settings.location.Lon, 'f', -1, 64))
	}

	parsedURL.RawQuery = query.Encode()

	return parsedURL, nil
}

// fetchServerListResponse performs the HTTP request for server list, handling fallback to alternative URL.
//
// The XML list is used when the JSON list comes back empty or with a status outside 2xx. An error status from the
// XML list is returned as a *StatusError.
//
// Parameters:
//   - ctx: cancellation and deadline for the requests.
//   - reqURL: the JSON list's URL.
//
// Returns:
//   - *http.Response: the response, whose body the caller closes.
//   - payloadType: the format of the response.
//   - error: the error from building or sending a request, or the XML list's status error.
func (s *Client) fetchServerListResponse(
	ctx context.Context,
	reqURL *url.URL,
) (*http.Response, payloadType, error) {
	s.log().Debug(ctx, "retrieving servers", logKeyURL, reqURL.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	payloadType := typeJSONPayload

	statusErr := checkStatus(resp)
	if statusErr != nil || resp.ContentLength == 0 {
		_ = resp.Body.Close()

		if statusErr != nil {
			s.log().
				Debug(ctx, "server list failed, trying the alternative list", logKeyError, statusErr)
		}

		alternativeURL, err := s.endpoint(serversAlternativePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to build alternative server list URL: %w", err)
		}

		req, err = http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			alternativeURL.String(),
			nil,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create alternative HTTP request: %w", err)
		}

		resp, err = s.doer.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to perform alternative HTTP request: %w", err)
		}

		err = checkStatus(resp)
		if err != nil {
			_ = resp.Body.Close()

			return nil, 0, fmt.Errorf("alternative server list request failed: %w", err)
		}

		payloadType = typeXMLPayload
	}

	return resp, payloadType, nil
}

// decodeServerList decodes the server list from the HTTP response based on payload type.
//
// Parameters:
//   - resp: the response to read.
//   - pType: the format of the response.
//
// Returns:
//   - Servers: the decoded servers.
//   - error: the decode error, or errPayloadDecode for an unknown format.
func (s *Client) decodeServerList(resp *http.Response, pType payloadType) (Servers, error) {
	var servers Servers

	switch pType {
	case typeJSONPayload:
		decoder := json.NewDecoder(resp.Body)

		err := decoder.Decode(&servers)
		if err != nil {
			return servers, fmt.Errorf("failed to decode JSON response: %w", err)
		}
	case typeXMLPayload:
		var list ServerList

		decoder := xml.NewDecoder(resp.Body)

		err := decoder.Decode(&list)
		if err != nil {
			return servers, fmt.Errorf("failed to decode XML response: %w", err)
		}

		servers = list.Servers
	default:
		return servers, errPayloadDecode
	}

	return servers, nil
}

// FetchServers retrieves the servers near the caller, or near the location set with [WithLocation], and probes the
// latency of each one.
//
// Each server's Latency is the round trip of one quick probe with the client's ping mode, or [PingTimeout] when the
// probe failed.
//
// Parameters:
//   - ctx: cancellation and deadline for the requests and the probes.
//
// Returns:
//   - Servers: the servers, sorted by distance.
//   - error: [ErrServerNotFound] for an empty list, an error wrapping [ErrUnexpectedStatus] when the lists answer
//     with a status outside 2xx, the context's error when it ended before the probes finished, or the request or
//     decoding error.
func (s *Client) FetchServers(ctx context.Context) (Servers, error) {
	if s == nil {
		return Servers{}, errSpeedtestClientNil
	}

	reqURL, err := s.buildServerListURL()
	if err != nil {
		return Servers{}, err
	}

	resp, payloadType, err := s.fetchServerListResponse(ctx, reqURL)
	if err != nil {
		return Servers{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	servers, err := s.decodeServerList(resp, payloadType)
	if err != nil {
		return servers, err
	}

	s.log().Debug(ctx, "pinging servers", logKeyCount, len(servers))

	s.pingServers(ctx, servers)

	// Probes cut short by the context all fail, which must not look like servers that never answered.
	err = ctx.Err()
	if err != nil {
		return Servers{}, fmt.Errorf("server probes ended by context: %w", err)
	}

	sortByDistance(servers, s.settings.location)

	if len(servers) == 0 {
		return servers, ErrServerNotFound
	}

	return servers, nil
}

// sortByDistance sorts servers by their distance from the point the list was requested for.
//
// speedtest.net reports each server's distance in whole kilometres from that point: the location set with
// WithLocation, or the caller's own. With a location, the distances are measured here to two decimals from the
// same point, which also fills them in for the alternative list, which has none. Servers whose coordinates do not
// parse cannot be measured, so they follow the measured ones. Whole kilometres often tie, so the sort is stable and
// keeps speedtest.net's order among equally distant servers.
//
// Parameters:
//   - servers: the servers, sorted in place.
//   - location: the location set with WithLocation, or nil.
func sortByDistance(servers Servers, location *Location) {
	if location == nil {
		sort.Stable(ByDistance{servers})

		return
	}

	unmeasured := make(map[*Server]bool)

	for _, server := range servers {
		serverLat, latErr := strconv.ParseFloat(server.Lat, 64)
		serverLon, lonErr := strconv.ParseFloat(server.Lon, 64)

		if latErr != nil || lonErr != nil {
			unmeasured[server] = true

			continue
		}

		server.Distance = distance(serverLat, serverLon, location.Lat, location.Lon)
	}

	slices.SortStableFunc(servers, func(first, second *Server) int {
		switch {
		case unmeasured[first] != unmeasured[second] && unmeasured[first]:
			return 1
		case unmeasured[first] != unmeasured[second]:
			return -1
		default:
			return cmp.Compare(first.Distance, second.Distance)
		}
	})
}

// distance returns the great-circle distance between two points with the haversine formula.
//
// Parameters:
//   - lat1: the first point's latitude in degrees.
//   - lon1: the first point's longitude in degrees.
//   - lat2: the second point's latitude in degrees.
//   - lon2: the second point's longitude in degrees.
//
// Returns:
//   - float64: the distance in kilometres.
func distance(lat1, lon1, lat2, lon2 float64) float64 {
	radius := 6378.137

	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0

	deltaPhiHalf := (lat1 - lat2) * math.Pi / 360.0
	deltaLambdaHalf := (lon1 - lon2) * math.Pi / 360.0
	sinePhiHalf2 := math.Sin(
		deltaPhiHalf,
	)*math.Sin(
		deltaPhiHalf,
	) + math.Cos(
		phi1,
	)*math.Cos(
		phi2,
	)*math.Sin(
		deltaLambdaHalf,
	)*math.Sin(
		deltaLambdaHalf,
	) // phi half-angle sine ^ 2
	delta := 2 * math.Atan2(
		math.Sqrt(sinePhiHalf2),
		math.Sqrt(1-sinePhiHalf2),
	) // 2 arc sine

	return radius * delta // r * delta
}

// FindServer finds server by serverID in given server list.
// If the id is not found in the given list, return the server with the lowest latency.
//
// Parameters:
//   - serverID: the IDs to find, or empty for the lowest latency server.
//
// Returns:
//   - Servers: the servers found, in the order of serverID.
//   - error: [ErrServerNotFound] when the list is empty.
func (servers Servers) FindServer(serverID []int) (Servers, error) {
	retServer := Servers{}

	if len(servers) == 0 {
		return retServer, ErrServerNotFound
	}

	for _, sid := range serverID {
		for _, s := range servers {
			id, _ := strconv.Atoi(s.ID)
			if sid == id {
				retServer = append(retServer, s)

				break
			}
		}
	}

	if len(retServer) == 0 {
		// choose the lowest latency server
		var (
			minLatency     int64 = math.MaxInt64
			minServerIndex int
		)

		for index, server := range servers {
			if server.Latency <= 0 {
				continue
			}

			if minLatency > server.Latency.Milliseconds() {
				minLatency = server.Latency.Milliseconds()
				minServerIndex = index
			}
		}

		retServer = append(retServer, servers[minServerIndex])
	}

	return retServer, nil
}

// String representation of ServerList.
//
// Returns:
//   - string: each server's [Server.String] line, joined.
func (servers ServerList) String() string {
	slr := ""

	var slrSb409 strings.Builder
	for _, server := range servers.Servers {
		slrSb409.WriteString(server.String())
	}

	slr += slrSb409.String()

	return slr
}

// String representation of Servers.
//
// Returns:
//   - string: each server's [Server.String] line, joined.
func (servers Servers) String() string {
	slr := ""

	var slrSb418 strings.Builder
	for _, server := range servers {
		slrSb418.WriteString(server.String())
	}

	slr += slrSb418.String()

	return slr
}

// String representation of Server.
//
// Returns:
//   - string: the server's ID, distance, city, country, and sponsor, such as
//     "[1001] 6.95km Tokyo (Japan) by Example Networks".
func (s *Server) String() string {
	if s == nil {
		return "<nil server>"
	}

	if s.Sponsor == "?" {
		return fmt.Sprintf("[%4s] %s", s.ID, s.Name)
	}

	if len(s.Country) == 0 {
		return fmt.Sprintf("[%4s] %.2fkm %s by %s", s.ID, s.Distance, s.Name, s.Sponsor)
	}

	return fmt.Sprintf("[%4s] %.2fkm %s (%s) by %s", s.ID, s.Distance, s.Name, s.Country, s.Sponsor)
}

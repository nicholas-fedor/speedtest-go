// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// API is a fake speedtest.net API and speedtest server backed by an [httptest.Server].
//
// It is safe for concurrent use, since clients ping every listed server in parallel.
type API struct {
	// server serves every endpoint.
	server *httptest.Server
	// responses maps an endpoint path to the response that replaces its normal behavior.
	responses map[string]Response
	// prefix is the path prefix of the speedtest.net endpoints.
	prefix string
	// serverHost overrides the host:port that listed servers report, or is empty to report the HTTP server.
	serverHost string
	// user is the caller identity reported by the user info and server lookup endpoints.
	user User
	// servers is the server list.
	servers []Server
	// requests records every request received, in order.
	requests []Request
	// downloadSize is the number of bytes each download request returns.
	downloadSize int64
	// mu guards responses and requests.
	mu sync.Mutex
}

// Option configures an [API] created by [NewAPI].
type Option func(*API)

// Response replaces an endpoint's normal behavior.
type Response struct {
	// Body is the response body. An empty body is sent with Content-Length 0.
	Body string
	// Location is the redirect target. Setting it without a Status sends 302 Found.
	Location string
	// Status is the HTTP status code. Zero means 200 OK, or 302 Found when Location is set.
	Status int
}

// Request is a request the fake received.
type Request struct {
	// Query is the parsed query string.
	Query url.Values
	// Header is a copy of the request headers.
	Header http.Header
	// Method is the HTTP method.
	Method string
	// Path is the request path as received, including any prefix.
	Path string
	// Endpoint is the endpoint the path matched, such as [PathServers], or empty for an unknown path.
	Endpoint string
	// ServerID is the listed server a per-server request was addressed to, or empty for the root server paths and
	// the speedtest.net endpoints.
	ServerID string
	// BodySize is the number of body bytes the fake read. Overridden responses read no body.
	BodySize int64
}

// Endpoint paths. The speedtest.net endpoints are served under the [WithPathPrefix] prefix. The per-server
// endpoints are served at these root paths for custom servers, and under /speedtest/<id>/ for each listed server,
// so requests can be attributed to the server they were sent to.
const (
	// PathUserConfig reports the caller's details as XML.
	PathUserConfig = "/speedtest-config.php"
	// PathServers lists servers as JSON.
	PathServers = "/api/js/servers"
	// PathServersStatic lists servers as XML.
	PathServersStatic = "/speedtest-servers-static.php"
	// PathServerLookup reports the caller and the server named by the serverid query parameter as XML.
	PathServerLookup = "/api/ios-config.php"
	// PathLatency answers latency probes.
	PathLatency = "/speedtest/latency.txt"
	// PathDownload stands for every /speedtest/randomNxN.jpg download, which all share one behavior.
	PathDownload = "/speedtest/random.jpg"
	// PathUpload accepts uploads. Its per-server form is each listed server's URL.
	PathUpload = "/speedtest/upload.php"
)

// DefaultDownloadSize is the number of bytes each download returns unless [WithDownloadSize] changes it.
// It is small so that download tests stay fast.
const DefaultDownloadSize = 64 << 10

// serverPathPrefix is the path prefix of the per-server endpoints.
const serverPathPrefix = "/speedtest/"

// Per-server file names, matched after the server path prefix and any server ID.
const (
	// latencyFile answers latency probes.
	latencyFile = "latency.txt"
	// uploadFile accepts uploads.
	uploadFile = "upload.php"
	// downloadPrefix starts every download file name, such as random1000x1000.jpg.
	downloadPrefix = "random"
	// downloadSuffix ends every download file name.
	downloadSuffix = ".jpg"
)

// apiEndpoints are the speedtest.net endpoints served under the path prefix.
var apiEndpoints = []string{PathUserConfig, PathServers, PathServersStatic, PathServerLookup}

// downloadChunk is the block written repeatedly to fill a download.
var downloadChunk = bytes.Repeat([]byte{0xAA}, 4<<10)

// NewAPI starts a fake API and closes it when the test ends.
//
// Parameters:
//   - tb: the test or benchmark that owns the server.
//   - opts: options that change the default user, servers, prefix, or download size.
//
// Returns:
//   - *API: the running fake.
func NewAPI(tb testing.TB, opts ...Option) *API {
	tb.Helper()

	api := &API{
		responses:    map[string]Response{},
		user:         DefaultUser(),
		servers:      DefaultServers(),
		downloadSize: DefaultDownloadSize,
	}

	for _, opt := range opts {
		opt(api)
	}

	api.server = httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	tb.Cleanup(api.server.Close)

	return api
}

// WithPathPrefix serves the speedtest.net endpoints under prefix, such as "/mirror".
//
// The prefix must not start with "/speedtest/", which the per-server endpoints use.
//
// Parameters:
//   - prefix: the path prefix, starting with a slash and without a trailing slash.
//
// Returns:
//   - Option: the option.
func WithPathPrefix(prefix string) Option {
	return func(api *API) {
		api.prefix = prefix
	}
}

// WithServerHost makes every listed server report hostPort as its host, such as the address of a [TCPServer], so
// that TCP pings and packet-loss tests reach it. Server URLs still point at the HTTP fake.
//
// Parameters:
//   - hostPort: the host:port that listed servers report.
//
// Returns:
//   - Option: the option.
func WithServerHost(hostPort string) Option {
	return func(api *API) {
		api.serverHost = hostPort
	}
}

// WithUser replaces the caller identity.
//
// Parameters:
//   - user: the caller identity to report.
//
// Returns:
//   - Option: the option.
func WithUser(user User) Option {
	return func(api *API) {
		api.user = user
	}
}

// WithServers replaces the server list.
//
// Parameters:
//   - servers: the servers to list, in response order.
//
// Returns:
//   - Option: the option.
func WithServers(servers ...Server) Option {
	return func(api *API) {
		api.servers = slices.Clone(servers)
	}
}

// WithDownloadSize sets the number of bytes each download returns.
//
// Parameters:
//   - size: the download size in bytes.
//
// Returns:
//   - Option: the option.
func WithDownloadSize(size int64) Option {
	return func(api *API) {
		api.downloadSize = size
	}
}

// URL returns the base URL for the speedtest.net endpoints, including any path prefix.
//
// Returns:
//   - string: the value to use as the client's base URL.
func (api *API) URL() string {
	return api.server.URL + api.prefix
}

// UploadURL returns the root upload URL, which works as a custom server URL.
//
// Requests sent through it have an empty [Request.ServerID].
//
// Returns:
//   - string: the upload URL.
func (api *API) UploadURL() string {
	return api.server.URL + PathUpload
}

// ServerUploadURL returns the upload URL of the listed server with the given ID.
//
// Requests derived from it, such as latency probes and downloads, are recorded with that [Request.ServerID].
//
// Parameters:
//   - id: the listed server's ID, which must be safe to use as a path segment.
//
// Returns:
//   - string: the server's upload URL.
func (api *API) ServerUploadURL(id string) string {
	return api.server.URL + serverPathPrefix + id + "/" + uploadFile
}

// Host returns the host and port that every listed server reports: the [WithServerHost] address when set,
// otherwise the HTTP fake's own address.
//
// Returns:
//   - string: the server's host:port.
func (api *API) Host() string {
	if api.serverHost != "" {
		return api.serverHost
	}

	return api.server.Listener.Addr().String()
}

// SetResponse replaces the behavior of an endpoint for every later request.
//
// Parameters:
//   - endpoint: one of the Path constants.
//   - resp: the response to send instead.
func (api *API) SetResponse(endpoint string, resp Response) {
	api.mu.Lock()
	defer api.mu.Unlock()

	api.responses[endpoint] = resp
}

// Requests returns a copy of every request received so far, in order.
//
// Returns:
//   - []Request: the recorded requests.
func (api *API) Requests() []Request {
	api.mu.Lock()
	defer api.mu.Unlock()

	return slices.Clone(api.requests)
}

// Requested reports whether any received request matched endpoint.
//
// Parameters:
//   - endpoint: one of the Path constants.
//
// Returns:
//   - bool: true when at least one request matched.
func (api *API) Requested(endpoint string) bool {
	return slices.ContainsFunc(api.Requests(), func(req Request) bool {
		return req.Endpoint == endpoint
	})
}

// serveHTTP routes a request to its endpoint, records it, and writes the response.
//
// The request is recorded before any response bytes are written, so a client that has read a response can rely on
// [API.Requests] including the request. Upload bodies are read first, so their size is recorded too.
//
// Parameters:
//   - w: the response writer.
//   - r: the request.
func (api *API) serveHTTP(w http.ResponseWriter, r *http.Request) {
	endpoint, serverID := api.route(r.URL.Path)

	api.mu.Lock()
	override, overridden := api.responses[endpoint]
	api.mu.Unlock()

	record := Request{
		Query:    r.URL.Query(),
		Header:   r.Header.Clone(),
		Method:   r.Method,
		Path:     r.URL.Path,
		Endpoint: endpoint,
		ServerID: serverID,
		BodySize: 0,
	}

	if !overridden && endpoint == PathUpload {
		record.BodySize, _ = io.Copy(io.Discard, r.Body)
	}

	api.mu.Lock()
	api.requests = append(api.requests, record)
	api.mu.Unlock()

	if overridden {
		writeOverride(w, override)

		return
	}

	switch endpoint {
	case PathUserConfig:
		writeXML(w, xmlSettings{XMLName: xml.Name{}, Client: toXMLClient(api.user), Servers: nil})
	case PathServers:
		api.writeServersJSON(w)
	case PathServersStatic:
		writeXML(w, xmlSettings{XMLName: xml.Name{}, Client: nil, Servers: api.xmlServers("")})
	case PathServerLookup:
		writeXML(w, xmlSettings{
			XMLName: xml.Name{},
			Client:  toXMLClient(api.user),
			Servers: api.xmlServers(r.URL.Query().Get("serverid")),
		})
	case PathLatency:
		writeBody(w, "text/plain", []byte("test=test"))
	case PathDownload:
		api.writeDownload(w)
	case PathUpload:
		writeBody(w, "text/plain", []byte("size="+strconv.FormatInt(record.BodySize, 10)))
	default:
		http.NotFound(w, r)
	}
}

// route maps a request path to the endpoint it serves and the listed server it was addressed to.
//
// Parameters:
//   - path: the request path.
//
// Returns:
//   - endpoint: one of the Path constants, or empty when the path matches no endpoint.
//   - serverID: the listed server's ID for a per-server path, or empty.
//
//nolint:nonamedreturns // Same-type returns need names.
func (api *API) route(path string) (endpoint, serverID string) {
	file, isServerPath := strings.CutPrefix(path, serverPathPrefix)
	if !isServerPath {
		rest, ok := strings.CutPrefix(path, api.prefix)
		if ok && slices.Contains(apiEndpoints, rest) {
			return rest, ""
		}

		return "", ""
	}

	if id, name, ok := strings.Cut(file, "/"); ok {
		if !api.hasServer(id) {
			return "", ""
		}

		serverID, file = id, name
	}

	switch {
	case file == latencyFile:
		return PathLatency, serverID
	case file == uploadFile:
		return PathUpload, serverID
	case strings.HasPrefix(file, downloadPrefix) && strings.HasSuffix(file, downloadSuffix):
		return PathDownload, serverID
	default:
		return "", ""
	}
}

// hasServer reports whether a listed server has the given ID.
//
// Parameters:
//   - id: the server ID.
//
// Returns:
//   - bool: true when the ID belongs to a listed server.
func (api *API) hasServer(id string) bool {
	return slices.ContainsFunc(api.servers, func(server Server) bool {
		return server.ID == id
	})
}

// writeServersJSON writes the server list as JSON.
//
// Parameters:
//   - w: the response writer.
func (api *API) writeServersJSON(w http.ResponseWriter) {
	list := make([]wireServer, 0, len(api.servers))
	for _, server := range api.servers {
		list = append(list, wireServer{
			URL:     api.ServerUploadURL(server.ID),
			Lat:     server.Lat,
			Lon:     server.Lon,
			Name:    server.Name,
			Country: server.Country,
			CC:      server.CC,
			Sponsor: server.Sponsor,
			ID:      server.ID,
			Host:    api.Host(),
		})
	}

	body, err := json.Marshal(list)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	writeBody(w, "application/json", body)
}

// xmlServers converts the server list to XML elements.
//
// Parameters:
//   - id: when not empty, only the server with this ID is included.
//
// Returns:
//   - []xmlServer: the server elements.
func (api *API) xmlServers(id string) []xmlServer {
	servers := make([]xmlServer, 0, len(api.servers))
	for _, server := range api.servers {
		if id != "" && server.ID != id {
			continue
		}

		servers = append(servers, xmlServer{
			URL:     api.ServerUploadURL(server.ID),
			Lat:     server.Lat,
			Lon:     server.Lon,
			Name:    server.Name,
			Country: server.Country,
			CC:      server.CC,
			Sponsor: server.Sponsor,
			ID:      server.ID,
			Host:    api.Host(),
		})
	}

	return servers
}

// writeDownload streams the configured number of download bytes.
//
// Parameters:
//   - w: the response writer.
func (api *API) writeDownload(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.FormatInt(api.downloadSize, 10))

	for remaining := api.downloadSize; remaining > 0; {
		chunk := downloadChunk[:min(remaining, int64(len(downloadChunk)))]

		written, err := w.Write(chunk)
		if err != nil {
			return
		}

		remaining -= int64(written)
	}
}

// writeOverride writes a response that replaces an endpoint's normal behavior.
//
// Parameters:
//   - w: the response writer.
//   - resp: the response to write.
func writeOverride(w http.ResponseWriter, resp Response) {
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
		if resp.Location != "" {
			status = http.StatusFound
		}
	}

	if resp.Location != "" {
		w.Header().Set("Location", resp.Location)
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(resp.Body)))
	w.WriteHeader(status)
	_, _ = io.WriteString(w, resp.Body)
}

// writeXML writes an XML document with its declaration.
//
// Parameters:
//   - w: the response writer.
//   - doc: the root element.
func writeXML(w http.ResponseWriter, doc xmlSettings) {
	body, err := xml.Marshal(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	writeBody(w, "application/xml", append([]byte(xml.Header), body...))
}

// writeBody writes a complete body with an explicit Content-Length, which clients use to detect an empty list.
//
// Parameters:
//   - w: the response writer.
//   - contentType: the Content-Type header value.
//   - body: the response body.
func writeBody(w http.ResponseWriter, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}

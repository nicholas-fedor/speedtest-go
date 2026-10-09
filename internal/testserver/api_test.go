// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package testserver

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noRedirectClient returns redirects to the caller instead of following them, so tests can inspect them.
var noRedirectClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// reply is the part of a response the tests inspect, read in full before the body is closed.
type reply struct {
	// header is the response headers.
	header http.Header
	// body is the full response body.
	body []byte
	// status is the HTTP status code.
	status int
	// contentLength is the declared Content-Length.
	contentLength int64
}

// do sends a request to the fake without following redirects and returns the read response.
func do(t *testing.T, method, rawURL, body string) reply {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(),
		method,
		rawURL,
		strings.NewReader(body),
	)
	require.NoError(t, err)

	resp, err := noRedirectClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return reply{
		header:        resp.Header,
		body:          data,
		status:        resp.StatusCode,
		contentLength: resp.ContentLength,
	}
}

// TestAPI_UserConfig checks that the user info endpoint reports the configured user as an XML client element.
func TestAPI_UserConfig(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	got := do(t, http.MethodGet, api.URL()+PathUserConfig, "")
	require.Equal(t, http.StatusOK, got.status)

	var doc xmlSettings
	require.NoError(t, xml.Unmarshal(got.body, &doc))
	require.NotNil(t, doc.Client)

	assert.Equal(t, toXMLClient(DefaultUser()), doc.Client)
	assert.Empty(t, doc.Servers)
}

// TestAPI_ServersJSON checks that the JSON list points every server back at the fake and keeps the listed order.
func TestAPI_ServersJSON(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	got := do(t, http.MethodGet, api.URL()+PathServers+"?search=osaka", "")
	require.Equal(t, http.StatusOK, got.status)
	assert.Equal(t, "application/json", got.header.Get("Content-Type"))

	var list []wireServer
	require.NoError(t, json.Unmarshal(got.body, &list))
	require.Len(t, list, 2)

	assert.Equal(t, "1002", list[0].ID)
	assert.Equal(t, "1001", list[1].ID)

	for _, server := range list {
		assert.Equal(t, api.ServerUploadURL(server.ID), server.URL)
		assert.Equal(t, api.Host(), server.Host)
	}

	requests := api.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "osaka", requests[0].Query.Get("search"))
}

// TestAPI_ServersStatic checks that the XML list carries every server and no client element.
func TestAPI_ServersStatic(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	got := do(t, http.MethodGet, api.URL()+PathServersStatic, "")

	var doc xmlSettings
	require.NoError(t, xml.Unmarshal(got.body, &doc))

	assert.Nil(t, doc.Client)
	require.Len(t, doc.Servers, 2)
	assert.Equal(t, api.ServerUploadURL(doc.Servers[0].ID), doc.Servers[0].URL)
}

// TestAPI_ServerLookup checks that the lookup returns only the requested server, and none for an unknown ID.
func TestAPI_ServerLookup(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	tests := []struct {
		name    string
		id      string
		wantIDs []string
	}{
		{name: "known ID", id: "1001", wantIDs: []string{"1001"}},
		{name: "unknown ID", id: "9999", wantIDs: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := do(t, http.MethodGet, api.URL()+PathServerLookup+"?serverid="+tt.id, "")

			var doc xmlSettings
			require.NoError(t, xml.Unmarshal(got.body, &doc))
			require.NotNil(t, doc.Client)

			var ids []string
			if len(doc.Servers) > 0 {
				ids = make([]string, 0, len(doc.Servers))
			}

			for _, server := range doc.Servers {
				ids = append(ids, server.ID)
			}

			assert.Equal(t, tt.wantIDs, ids)
		})
	}
}

// TestAPI_ServerEndpoints checks the latency, download, and upload endpoints, and that uploads record their size.
func TestAPI_ServerEndpoints(t *testing.T) {
	t.Parallel()

	api := NewAPI(t, WithDownloadSize(10_000))

	latency := do(t, http.MethodGet, api.URL()+PathLatency, "")
	assert.Equal(t, "test=test", string(latency.body))

	download := do(t, http.MethodGet, api.URL()+"/speedtest/random1000x1000.jpg", "")
	assert.Equal(t, http.StatusOK, download.status)
	assert.Len(t, download.body, 10_000)

	upload := do(t, http.MethodPost, api.UploadURL(), strings.Repeat("x", 1234))
	assert.Equal(t, "size=1234", string(upload.body))

	requests := api.Requests()
	require.Len(t, requests, 3)
	assert.Equal(t, PathDownload, requests[1].Endpoint)
	assert.Equal(t, "/speedtest/random1000x1000.jpg", requests[1].Path)
	assert.Equal(t, int64(1234), requests[2].BodySize)

	for _, req := range requests {
		assert.Empty(t, req.ServerID, "root server paths belong to no listed server")
	}
}

// TestAPI_PerServerPaths checks that requests under a listed server's path are attributed to that server, and that
// unknown servers and unknown files are rejected.
func TestAPI_PerServerPaths(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)
	root := strings.TrimSuffix(api.ServerUploadURL("1001"), "/upload.php")

	tests := []struct {
		name         string
		method       string
		path         string
		wantEndpoint string
		wantServerID string
		wantStatus   int
	}{
		{
			name:         "latency",
			method:       http.MethodGet,
			path:         root + "/latency.txt",
			wantEndpoint: PathLatency,
			wantServerID: "1001",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "download",
			method:       http.MethodGet,
			path:         root + "/random350x350.jpg",
			wantEndpoint: PathDownload,
			wantServerID: "1001",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "upload",
			method:       http.MethodPost,
			path:         api.ServerUploadURL("1001"),
			wantEndpoint: PathUpload,
			wantServerID: "1001",
			wantStatus:   http.StatusOK,
		},
		{
			name:       "unknown server",
			method:     http.MethodGet,
			path:       strings.Replace(root, "/1001", "/9999", 1) + "/latency.txt",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "nested file",
			method:     http.MethodGet,
			path:       root + "/extra/latency.txt",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := do(t, tt.method, tt.path, "payload")
			assert.Equal(t, tt.wantStatus, got.status)

			var found bool

			for _, req := range api.Requests() {
				if strings.HasSuffix(tt.path, req.Path) && req.Method == tt.method {
					found = true

					assert.Equal(t, tt.wantEndpoint, req.Endpoint)
					assert.Equal(t, tt.wantServerID, req.ServerID)
				}
			}

			assert.True(t, found, "the request should be recorded")
		})
	}
}

// TestAPI_PathPrefix checks that the prefix applies only to the speedtest.net endpoints.
func TestAPI_PathPrefix(t *testing.T) {
	t.Parallel()

	api := NewAPI(t, WithPathPrefix("/mirror"))

	assert.True(t, strings.HasSuffix(api.URL(), "/mirror"))

	got := do(t, http.MethodGet, api.URL()+PathUserConfig, "")
	assert.Equal(t, http.StatusOK, got.status)

	unprefixed := strings.TrimSuffix(api.URL(), "/mirror")

	got = do(t, http.MethodGet, unprefixed+PathUserConfig, "")
	assert.Equal(t, http.StatusNotFound, got.status, "an unprefixed API path must not match")

	got = do(t, http.MethodGet, unprefixed+PathLatency, "")
	assert.Equal(t, http.StatusOK, got.status, "server endpoints stay at the root")

	requests := api.Requests()
	require.Len(t, requests, 3)
	assert.Equal(t, "/mirror"+PathUserConfig, requests[0].Path)
	assert.Equal(t, PathUserConfig, requests[0].Endpoint)
	assert.Empty(t, requests[1].Endpoint)
}

// TestAPI_UnknownPaths checks that paths outside the known endpoints answer 404 and record no endpoint.
func TestAPI_UnknownPaths(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	for _, path := range []string{"/", "/speedtest/other.txt", "/speedtest/random.png", "/api/js"} {
		got := do(t, http.MethodGet, api.URL()+path, "")
		assert.Equal(t, http.StatusNotFound, got.status, path)
	}

	assert.False(t, api.Requested(PathServers))
}

// TestAPI_SetResponse checks status, redirect, and empty-body overrides, and that overrides read no upload body.
func TestAPI_SetResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		endpoint     string
		method       string
		override     Response
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		{
			name:       "error status",
			endpoint:   PathLatency,
			method:     http.MethodGet,
			override:   Response{Status: http.StatusInternalServerError, Body: "broken"},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "broken",
		},
		{
			name:         "redirect defaults to 302",
			endpoint:     PathUpload,
			method:       http.MethodPost,
			override:     Response{Location: "https://upload.example/speedtest/upload.php"},
			wantStatus:   http.StatusFound,
			wantLocation: "https://upload.example/speedtest/upload.php",
		},
		{
			name:       "empty body defaults to 200",
			endpoint:   PathServers,
			method:     http.MethodGet,
			override:   Response{},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := NewAPI(t)
			api.SetResponse(tt.endpoint, tt.override)

			got := do(t, tt.method, api.URL()+tt.endpoint, "payload")

			assert.Equal(t, tt.wantStatus, got.status)
			assert.Equal(t, tt.wantLocation, got.header.Get("Location"))
			assert.Equal(t, tt.wantBody, string(got.body))
			assert.Equal(t, int64(len(tt.wantBody)), got.contentLength)

			requests := api.Requests()
			require.Len(t, requests, 1)
			assert.Zero(t, requests[0].BodySize)
			assert.True(t, api.Requested(tt.endpoint))
		})
	}
}

// TestAPI_Options checks that the user and server options replace the defaults.
func TestAPI_Options(t *testing.T) {
	t.Parallel()

	user := User{IP: "198.51.100.1", Lat: "1", Lon: "2", ISP: "Other ISP", Country: "GB"}
	server := Server{
		ID:      "7",
		Name:    "London",
		Country: "United Kingdom",
		CC:      "GB",
		Sponsor: "S",
		Lat:     "51",
		Lon:     "0",
	}

	api := NewAPI(t, WithUser(user), WithServers(server))

	got := do(t, http.MethodGet, api.URL()+PathServerLookup+"?serverid=7", "")

	var doc xmlSettings
	require.NoError(t, xml.Unmarshal(got.body, &doc))

	assert.Equal(t, toXMLClient(user), doc.Client)
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "London", doc.Servers[0].Name)
	assert.Equal(t, "GB", doc.Servers[0].CC)
}

// TestAPI_RecordsHeaders checks that request headers are recorded, so tests can assert on the User-Agent.
func TestAPI_RecordsHeaders(t *testing.T) {
	t.Parallel()

	api := NewAPI(t)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		api.URL()+PathLatency,
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "speedtest-go-test")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	requests := api.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, "speedtest-go-test", requests[0].Header.Get("User-Agent"))
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/transport"
)

// TestTCPHost checks the TCP address taken from a server's Host and URL fields.
func TestTCPHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		server  Server
		want    string
		wantErr error
	}{
		{
			name:   "host with port",
			server: Server{Host: "speed.example:8080"},
			want:   "speed.example:8080",
		},
		{
			name:   "host without port",
			server: Server{Host: "speed.example"},
			want:   "speed.example:8080",
		},
		{
			name:   "IPv6 host with port",
			server: Server{Host: "[2001:db8::1]:8080"},
			want:   "[2001:db8::1]:8080",
		},
		{
			name:   "bracketed IPv6 host",
			server: Server{Host: "[2001:db8::1]"},
			want:   "[2001:db8::1]:8080",
		},
		{
			name:   "URL fallback",
			server: Server{URL: "http://speed.example/speedtest/upload.php"},
			want:   "speed.example:8080",
		},
		{name: "no host", server: Server{ID: "1001"}, wantErr: errNoTCPHost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tcpHost(&tt.server)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestTransferKinds checks that each test mode selects its own request functions, and that only HTTP uploads
// resolve upload redirects.
func TestTransferKinds(t *testing.T) {
	t.Parallel()

	pointer := func(fn any) uintptr { return reflect.ValueOf(fn).Pointer() }

	httpClient := newTestClient(t)
	assert.Equal(t, pointer(downloadRequest), pointer(httpClient.downloadKind().request))
	assert.Equal(t, pointer(uploadRequest), pointer(httpClient.uploadKind().request))
	assert.False(t, httpClient.downloadKind().resolveUploads)
	assert.True(t, httpClient.uploadKind().resolveUploads)

	tcpClient := newTestClient(t, WithProtocol(TCPTest))
	assert.Equal(t, pointer(tcpDownloadRequest), pointer(tcpClient.downloadKind().request))
	assert.Equal(t, pointer(tcpUploadRequest), pointer(tcpClient.uploadKind().request))
}

// uploadURLWatcher is an HTTP server standing in for a listed upload URL, counting the requests it receives.
func uploadURLWatcher(t *testing.T) (string, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32

	watcher := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(watcher.Close)

	return watcher.URL + "/speedtest/upload.php", &requests
}

// tcpClient returns a client in TCP test mode with a short capture time.
//
// Parameters:
//   - t: the test.
//
// Returns:
//   - *Client: the client.
func tcpClient(t *testing.T) *Client {
	t.Helper()

	return newTestClient(
		t,
		WithProtocol(TCPTest),
		WithConnections(2),
		WithTestDuration(300*time.Millisecond),
	)
}

// TestClient_TCPTests checks single-server download and upload tests in TCP mode: both measure a rate over the
// TCP protocol, and the listed HTTP upload URL is never contacted.
func TestClient_TCPTests(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	uploadURL, httpRequests := uploadURLWatcher(t)
	client := tcpClient(t)
	server := &Server{ID: "1001", Host: srv.Addr(), URL: uploadURL}

	download, err := client.Download(context.Background(), server)
	require.NoError(t, err)
	assert.Positive(t, float64(download.Rate))
	assert.Positive(t, download.Bytes)

	upload, err := client.Upload(context.Background(), server)
	require.NoError(t, err)
	assert.Positive(t, float64(upload.Rate))
	assert.Positive(t, upload.Bytes)

	assert.Positive(t, countCommands(srv, "DOWNLOAD "+strconv.Itoa(dlSizes[3]*1000)))
	assert.Positive(t, countCommands(srv, "UPLOAD "+strconv.Itoa(ulSizes[4]*1000)+" 0"))
	assert.Zero(t, httpRequests.Load(), "TCP tests must not resolve the HTTP upload URL")
}

// TestClient_TCPMultiTests checks that multi-server tests in TCP mode use every server in the pool.
func TestClient_TCPMultiTests(t *testing.T) {
	t.Parallel()

	first, second := testserver.NewTCPServer(t), testserver.NewTCPServer(t)
	uploadURL, httpRequests := uploadURLWatcher(t)
	client := tcpClient(t)

	pool := Servers{
		{ID: "1001", Host: first.Addr(), URL: uploadURL},
		{ID: "1002", Host: second.Addr(), URL: uploadURL},
	}

	download, err := client.DownloadMulti(context.Background(), pool[0], pool)
	require.NoError(t, err)

	upload, err := client.UploadMulti(context.Background(), pool[0], pool)
	require.NoError(t, err)

	for _, srv := range []*testserver.TCPServer{first, second} {
		assert.Positive(t, countPrefix(srv, "DOWNLOAD "))
		assert.Positive(t, countPrefix(srv, "UPLOAD "))
	}

	assert.Positive(t, float64(download.Rate))
	assert.Positive(t, float64(upload.Rate))
	assert.Zero(t, httpRequests.Load(), "TCP tests must not resolve the HTTP upload URL")
}

// tcpRequestTo returns a request to a TCP server, with the smallest transfer size.
//
// Parameters:
//   - t: the test.
//   - host: the server's host and port, or empty for none.
//
// Returns:
//   - *transferRequest: the request.
func tcpRequestTo(t *testing.T, host string) *transferRequest {
	t.Helper()

	return &transferRequest{
		client:    newTestClient(t),
		server:    &Server{Host: host},
		uploadURL: "",
		size:      0,
	}
}

// countPrefix returns how many commands the fake server received starting with prefix.
func countPrefix(srv *testserver.TCPServer, prefix string) int {
	var count int

	for _, command := range srv.Commands() {
		if strings.HasPrefix(command.Line, prefix) {
			count++
		}
	}

	return count
}

// Test_tcpUploadRequest checks that an upload counts exactly the confirmed payload, without the command overhead.
func Test_tcpUploadRequest(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	request := &transferRequest{
		client:    newTestClient(t),
		server:    &Server{Host: srv.Addr()},
		uploadURL: "",
		size:      0,
	}
	connection := uploadConnection()

	require.NoError(t, tcpUploadRequest(context.Background(), request, connection))

	payload, err := transport.UploadPayloadSize(int64(ulSizes[0]) * 1000)
	require.NoError(t, err)
	assert.Equal(t, payload, connection.Bytes())
	assert.Equal(t, connection.Sent(), connection.Bytes(), "every payload byte read was confirmed")
}

// Test_tcpDownloadRequest checks a complete download, and that a truncated one is an error.
func Test_tcpDownloadRequest(t *testing.T) {
	t.Parallel()

	t.Run("complete", func(t *testing.T) {
		t.Parallel()

		srv := testserver.NewTCPServer(t)
		connection := downloadConnection()

		require.NoError(
			t,
			tcpDownloadRequest(context.Background(), tcpRequestTo(t, srv.Addr()), connection),
		)
		assert.Equal(t, int64(dlSizes[0])*1000, connection.Bytes())
	})

	t.Run("truncated", func(t *testing.T) {
		t.Parallel()

		srv := testserver.NewTCPServer(t)
		srv.SetFaults(testserver.Faults{TruncateDownloads: true})

		err := tcpDownloadRequest(
			context.Background(),
			tcpRequestTo(t, srv.Addr()),
			downloadConnection(),
		)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("no host", func(t *testing.T) {
		t.Parallel()

		err := tcpDownloadRequest(context.Background(), tcpRequestTo(t, ""), downloadConnection())
		require.ErrorIs(t, err, errNoTCPHost)
	})
}

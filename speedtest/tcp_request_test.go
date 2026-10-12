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
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
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

// TestRequestFor checks that each test mode selects its own request functions.
func TestRequestFor(t *testing.T) {
	t.Parallel()

	pointer := func(fn any) uintptr { return reflect.ValueOf(fn).Pointer() }

	httpServer := &Server{Context: New()}
	assert.Equal(t, pointer(downloadRequest), pointer(httpServer.downloadRequestFor()))
	assert.Equal(t, pointer(uploadRequest), pointer(httpServer.uploadRequestFor()))

	tcpServer := &Server{Context: New(WithUserConfig(&UserConfig{TestMode: TCPTest}))}
	assert.Equal(t, pointer(tcpDownloadRequest), pointer(tcpServer.downloadRequestFor()))
	assert.Equal(t, pointer(tcpUploadRequest), pointer(tcpServer.uploadRequestFor()))
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
func tcpClient() *Speedtest {
	client := New(WithUserConfig(&UserConfig{TestMode: TCPTest, MaxConnections: 2}))
	client.SetCaptureTime(300 * time.Millisecond)

	return client
}

// TestServer_TCPTests checks single-server download and upload tests in TCP mode: both measure a rate over the
// TCP protocol, and the listed HTTP upload URL is never contacted.
func TestServer_TCPTests(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	uploadURL, httpRequests := uploadURLWatcher(t)
	client := tcpClient()
	server := &Server{ID: "1001", Host: srv.Addr(), URL: uploadURL, Context: client}

	require.NoError(t, server.DownloadTestContext(context.Background()))
	assert.Positive(t, float64(server.DLSpeed))
	assert.Positive(t, client.GetTotalDownload())

	require.NoError(t, server.UploadTestContext(context.Background()))
	assert.Positive(t, float64(server.ULSpeed))
	assert.Positive(t, client.GetTotalUpload())

	assert.Positive(t, countCommands(srv, "DOWNLOAD "+strconv.Itoa(dlSizes[3]*1000)))
	assert.Positive(t, countCommands(srv, "UPLOAD "+strconv.Itoa(ulSizes[4]*1000)+" 0"))
	assert.Zero(t, httpRequests.Load(), "TCP tests must not resolve the HTTP upload URL")
}

// TestServer_TCPMultiTests checks that multi-server tests in TCP mode use every server in the pool.
func TestServer_TCPMultiTests(t *testing.T) {
	t.Parallel()

	first, second := testserver.NewTCPServer(t), testserver.NewTCPServer(t)
	uploadURL, httpRequests := uploadURLWatcher(t)
	client := tcpClient()

	pool := Servers{
		{ID: "1001", Host: first.Addr(), URL: uploadURL, Context: client},
		{ID: "1002", Host: second.Addr(), URL: uploadURL, Context: client},
	}

	require.NoError(t, pool[0].MultiDownloadTestContext(context.Background(), pool))
	require.NoError(t, pool[0].MultiUploadTestContext(context.Background(), pool))

	for _, srv := range []*testserver.TCPServer{first, second} {
		assert.Positive(t, countPrefix(srv, "DOWNLOAD "))
		assert.Positive(t, countPrefix(srv, "UPLOAD "))
	}

	assert.Positive(t, float64(pool[0].DLSpeed))
	assert.Positive(t, float64(pool[0].ULSpeed))
	assert.Zero(t, httpRequests.Load(), "TCP tests must not resolve the HTTP upload URL")
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
	client, manager := runningClient(t)
	server := &Server{Host: srv.Addr(), Context: client}

	require.NoError(t, tcpUploadRequest(context.Background(), server, 0))

	payload, err := transport.UploadPayloadSize(int64(ulSizes[0]) * 1000)
	require.NoError(t, err)
	assert.Equal(t, payload, manager.GetTotalUpload())
	assert.Zero(t, manager.GetUploadBacklog(), "every payload byte read was confirmed")
}

// Test_tcpDownloadRequest checks a complete download, and that a truncated one is an error.
func Test_tcpDownloadRequest(t *testing.T) {
	t.Parallel()

	t.Run("complete", func(t *testing.T) {
		t.Parallel()

		srv := testserver.NewTCPServer(t)
		client, manager := runningClient(t)

		require.NoError(
			t,
			tcpDownloadRequest(context.Background(), &Server{Host: srv.Addr(), Context: client}, 0),
		)
		assert.Equal(t, int64(dlSizes[0])*1000, manager.GetTotalDownload())
	})

	t.Run("truncated", func(t *testing.T) {
		t.Parallel()

		srv := testserver.NewTCPServer(t)
		srv.SetFaults(testserver.Faults{TruncateDownloads: true})

		client, _ := runningClient(t)

		err := tcpDownloadRequest(
			context.Background(),
			&Server{Host: srv.Addr(), Context: client},
			0,
		)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("no host", func(t *testing.T) {
		t.Parallel()

		client, _ := runningClient(t)

		err := tcpDownloadRequest(context.Background(), &Server{Context: client}, 0)
		require.ErrorIs(t, err, errNoTCPHost)
	})
}

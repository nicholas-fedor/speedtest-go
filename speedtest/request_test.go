// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/meter"
)

// requestTo returns a request from a client to a server at a URL, with the smallest transfer size.
//
// Parameters:
//   - client: the client that sends the request.
//   - url: the server's upload URL.
//
// Returns:
//   - *transferRequest: the request.
func requestTo(client *Client, url string) *transferRequest {
	return &transferRequest{client: client, server: &Server{URL: url}, uploadURL: "", size: 0}
}

// downloadConnection returns a connection outside a test run that counts a download request's bytes.
//
// Returns:
//   - *meter.Connection: the connection.
func downloadConnection() *meter.Connection {
	return meter.New(meter.Config{}).DownloadConnection()
}

// uploadConnection returns a connection outside a test run that counts an upload request's bytes.
//
// Returns:
//   - *meter.Connection: the connection.
func uploadConnection() *meter.Connection {
	return meter.New(meter.Config{}).UploadConnection()
}

// Test_downloadRequest_FakeServer checks that a successful download returns no error and counts its bytes, and that
// an error status returns a status error without counting the error page.
func Test_downloadRequest_FakeServer(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		connection := downloadConnection()

		require.NoError(
			t,
			downloadRequest(
				context.Background(),
				requestTo(newTestClient(t), api.ServerUploadURL("1001")),
				connection,
			),
		)

		assert.Equal(t, int64(testserver.DefaultDownloadSize), connection.Bytes())
	})

	t.Run("error status", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(
			testserver.PathDownload,
			testserver.Response{Status: http.StatusInternalServerError, Body: "error page"},
		)

		connection := downloadConnection()

		err := downloadRequest(
			context.Background(),
			requestTo(newTestClient(t), api.ServerUploadURL("1001")),
			connection,
		)
		require.ErrorIs(t, err, ErrUnexpectedStatus)

		assert.Zero(
			t,
			connection.Bytes(),
			"the error page must not count as downloaded data",
		)
	})
}

// failingBody is a response body whose reads fail.
type failingBody struct{}

// Read reports a connection reset.
func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

// Close does nothing.
func (failingBody) Close() error { return nil }

// roundTripFunc adapts a function to [http.RoundTripper].
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// Test_uploadRequest_FakeServer checks that an upload counts its bytes only once the server confirms the request
// with a successful response after taking the whole body, and that an error status, a redirect, a response before
// the body was read, or an unreadable response counts nothing.
func Test_uploadRequest_FakeServer(t *testing.T) {
	t.Parallel()

	wantSize := int64(ulSizes[0]*100-51) * 10

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		request := requestTo(newTestClient(t), api.ServerUploadURL("1001"))
		connection := uploadConnection()

		require.NoError(t, uploadRequest(context.Background(), request, connection))
		assert.Equal(t, wantSize, connection.Bytes())

		require.NoError(t, uploadRequest(context.Background(), request, connection))
		assert.Equal(t, 2*wantSize, connection.Bytes())
	})

	t.Run("error status", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		api.SetResponse(
			testserver.PathUpload,
			testserver.Response{Status: http.StatusInternalServerError, Body: "error page"},
		)

		connection := uploadConnection()

		err := uploadRequest(
			context.Background(),
			requestTo(newTestClient(t), api.ServerUploadURL("1001")),
			connection,
		)
		require.ErrorIs(t, err, ErrUnexpectedStatus)
		assert.Zero(
			t,
			connection.Bytes(),
			"a rejected upload must not count as uploaded data",
		)
	})

	t.Run("response before the body is read", func(t *testing.T) {
		t.Parallel()

		for _, read := range []int64{0, wantSize / 2} {
			client := newTestClient(t)
			connection := uploadConnection()
			client.doer = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					_, _ = io.CopyN(io.Discard, req.Body, read)

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       http.NoBody,
						Request:    req,
					}, nil
				}),
			}
			err := uploadRequest(
				context.Background(),
				requestTo(client, "http://upload.invalid/speedtest/upload.php"),
				connection,
			)
			require.ErrorIs(t, err, errUploadIncomplete)
			assert.Zero(
				t,
				connection.Bytes(),
				"an upload the server did not take must not count",
			)
		}
	})

	t.Run("redirect", func(t *testing.T) {
		t.Parallel()

		var followed atomic.Bool

		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/moved" {
				followed.Store(true)

				return
			}

			_, _ = io.Copy(io.Discard, r.Body)
			http.Redirect(w, r, "/moved", http.StatusFound)
		}))
		t.Cleanup(target.Close)

		connection := uploadConnection()

		err := uploadRequest(
			context.Background(),
			requestTo(newTestClient(t), target.URL+"/speedtest/upload.php"),
			connection,
		)
		require.ErrorIs(t, err, ErrUnexpectedStatus)
		assert.False(t, followed.Load(), "the redirect must not be followed with a GET")
		assert.Zero(t, connection.Bytes(), "a redirected upload must not count")
	})

	t.Run("unreadable response", func(t *testing.T) {
		t.Parallel()

		client := newTestClient(t)
		connection := uploadConnection()
		client.doer = &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				_, _ = io.Copy(io.Discard, req.Body)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       failingBody{},
					Request:    req,
				}, nil
			}),
		}
		err := uploadRequest(
			context.Background(),
			requestTo(client, "http://upload.invalid/speedtest/upload.php"),
			connection,
		)
		require.ErrorContains(t, err, "connection reset")
		assert.Zero(
			t,
			connection.Bytes(),
			"an unconfirmed upload must not count as uploaded data",
		)
	})
}

// redirector is a server that answers every request with a redirect to one target, like an http upload URL that
// moves to an https host. It records the methods it receives.
type redirector struct {
	server  *httptest.Server
	mu      sync.Mutex
	methods []string
}

// newRedirector starts a redirector that sends every request to target with status.
func newRedirector(t *testing.T, target string, status int) *redirector {
	t.Helper()

	r := &redirector{}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.methods = append(r.methods, req.Method)
		r.mu.Unlock()

		// Like the servers this works around, answer without reading an upload body.
		http.Redirect(w, req, target, status)
	}))
	t.Cleanup(r.server.Close)

	return r
}

// URL returns the redirecting upload URL.
func (r *redirector) URL() string {
	return r.server.URL + "/speedtest/upload.php"
}

// Methods returns the methods received so far.
func (r *redirector) Methods() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.methods)
}

// TestClient_resolveUploadURL checks that each redirect status is resolved with a single HEAD request, that the
// upload then goes straight to the final URL and counts, and that the server's listed URL is unchanged.
func TestClient_resolveUploadURL(t *testing.T) {
	t.Parallel()

	wantSize := int64(ulSizes[0]*100-51) * 10

	for _, status := range []int{
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			final := api.ServerUploadURL("1001")
			moved := newRedirector(t, final, status)

			client := newTestClient(t)
			request := requestTo(client, moved.URL())
			request.uploadURL = client.resolveUploadURL(context.Background(), request.server)

			assert.Equal(t, final, request.uploadTarget())
			assert.Equal(t, moved.URL(), request.server.URL, "the listed URL must stay as it was")

			connection := uploadConnection()
			require.NoError(t, uploadRequest(context.Background(), request, connection))
			assert.Equal(t, wantSize, connection.Bytes())
			assert.Equal(
				t,
				[]string{http.MethodHead},
				moved.Methods(),
				"only the HEAD may reach the old URL",
			)
		})
	}
}

// TestClient_resolveUploadURLFallsBack checks that uploads use the listed URL when nothing redirects and when the
// HEAD request fails.
func TestClient_resolveUploadURLFallsBack(t *testing.T) {
	t.Parallel()

	t.Run("no redirect", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		request := requestTo(newTestClient(t), api.ServerUploadURL("1001"))

		request.uploadURL = request.client.resolveUploadURL(context.Background(), request.server)

		assert.Empty(t, request.uploadURL)
		assert.Equal(t, request.server.URL, request.uploadTarget())
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()

		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()

		request := requestTo(newTestClient(t), closed.URL+"/speedtest/upload.php")

		request.uploadURL = request.client.resolveUploadURL(context.Background(), request.server)

		assert.Empty(t, request.uploadURL)
		assert.Equal(t, request.server.URL, request.uploadTarget())
	})
}

// TestClient_resolveUploadURLs checks that every server is resolved while no more than the limit of HEAD requests
// are in flight at once.
func TestClient_resolveUploadURLs(t *testing.T) {
	t.Parallel()

	const limit = 2

	api := testserver.NewAPI(t)

	var inFlight, maxInFlight atomic.Int32

	moved := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		current := inFlight.Add(1)
		defer inFlight.Add(-1)

		for {
			highest := maxInFlight.Load()
			if current <= highest || maxInFlight.CompareAndSwap(highest, current) {
				break
			}
		}

		// Hold each request briefly, so requests beyond the limit would overlap.
		time.Sleep(20 * time.Millisecond)

		http.Redirect(
			w,
			req,
			api.ServerUploadURL(strings.TrimPrefix(req.URL.Path, "/")),
			http.StatusFound,
		)
	}))
	t.Cleanup(moved.Close)

	servers := make(Servers, 0, 6)

	for index := range 6 {
		servers = append(servers, &Server{URL: fmt.Sprintf("%s/%d", moved.URL, index)})
	}

	resolved := newTestClient(t).resolveUploadURLs(context.Background(), servers, limit)

	assert.LessOrEqual(t, maxInFlight.Load(), int32(limit))

	for index := range servers {
		assert.Equal(t, api.ServerUploadURL(strconv.Itoa(index)), resolved[index])
	}
}

// TestUploadRequestClearsBacklog checks that a confirmed upload to the fake server sent exactly what it confirmed.
func TestUploadRequestClearsBacklog(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	connection := uploadConnection()

	require.NoError(
		t,
		uploadRequest(
			context.Background(),
			requestTo(newTestClient(t), api.ServerUploadURL("1001")),
			connection,
		),
	)

	assert.Positive(t, connection.Bytes())
	assert.Equal(t, connection.Sent(), connection.Bytes(), "every byte sent was confirmed")
}

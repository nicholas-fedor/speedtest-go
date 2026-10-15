// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/meter"
)

// requestFunc sends one download or upload request of a transfer, counting its data through the worker's
// connection.
type requestFunc func(ctx context.Context, request *transferRequest, connection *meter.Connection) error

// transferRequest is what a transfer's requests to one server need.
type transferRequest struct {
	// client sends the requests.
	client *Client
	// server is the server the requests go to.
	server *Server
	// uploadURL is where uploads go once redirects are resolved, or empty to use the server's URL.
	uploadURL string
	// size is the index into the download or upload size table.
	size int
}

// countingReader counts the bytes read through it. The HTTP transport reads a request body on its own goroutine,
// so the count is atomic.
type countingReader struct {
	// reader is the wrapped reader.
	reader io.Reader
	// count is the number of bytes read so far.
	count atomic.Int64
}

// Limits on resolving upload redirects.
const (
	// uploadResolveTimeout bounds the HEAD request that resolves upload redirects, so an unresponsive server cannot
	// hold up the start of a test.
	uploadResolveTimeout = 5 * time.Second
	// maxConcurrentUploadResolves bounds how many HEAD requests a multi-server test sends at once, since its pool can
	// be the whole fetched server list.
	maxConcurrentUploadResolves = 8
)

// Transfer sizes.
var (
	// dlSizes are the sides of the images downloads request, or their size in kB over TCP.
	dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	// ulSizes are the upload sizes in kB.
	ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000}
)

// Errors of server tests and requests.
var (
	// ErrConnectTimeout is returned when server connection times out.
	ErrConnectTimeout = errors.New("server connect timeout")
	// ErrUninitializedManager is returned when a test runs on a client that was not built by [New].
	ErrUninitializedManager = errors.New("uninitialized manager")
	// ErrServerNil is returned when the server is nil.
	ErrServerNil = errors.New("server is nil")
	// ErrNoAvailableServers is returned when no available servers are found.
	ErrNoAvailableServers = errors.New("not found available servers")
	// errNoHost reports a server URL that parses but names no host, such as a relative path.
	errNoHost = errors.New("server URL has no host")
	// errUploadIncomplete reports a successful upload response that arrived before the server took the whole body.
	errUploadIncomplete = errors.New("upload response arrived before the whole body was sent")
)

// Read reads from the wrapped reader and adds the bytes read to the count.
//
// Parameters:
//   - buffer: the destination.
//
// Returns:
//   - int: the number of bytes read.
//   - error: the wrapped reader's error.
func (r *countingReader) Read(buffer []byte) (int, error) {
	read, err := r.reader.Read(buffer)
	r.count.Add(int64(read))

	return read, err //nolint:wrapcheck // io.Reader callers compare the error with io.EOF.
}

// uploadTarget returns the URL uploads are sent to.
//
// Returns:
//   - string: the resolved upload URL, or the server's URL when nothing was resolved.
func (r *transferRequest) uploadTarget() string {
	if r.uploadURL != "" {
		return r.uploadURL
	}

	return r.server.URL
}

// downloadRequest downloads one image from the server over HTTP, counting each byte as it arrives.
//
// Parameters:
//   - ctx: cancellation for the request.
//   - request: the server and the download size.
//   - connection: the worker's connection, which counts the bytes.
//
// Returns:
//   - error: the request, status, or transfer error.
func downloadRequest(
	ctx context.Context,
	request *transferRequest,
	connection *meter.Connection,
) error {
	size := dlSizes[request.size]

	u, err := url.Parse(request.server.URL)
	if err != nil {
		return fmt.Errorf("failed to parse download URL: %w", err)
	}

	u.Path = path.Dir(u.Path)
	xdlURL := u.JoinPath(fmt.Sprintf("random%dx%d.jpg", size, size)).String()
	request.client.log().Debug(ctx, "downloading", logKeyURL, xdlURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xdlURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download HTTP request: %w", err)
	}

	resp, err := request.client.doer.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform download HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	// An error page must not count as downloaded data, so check before handing the body to a chunk.
	err = checkStatus(resp)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}

	err = connection.NewChunk().DownloadHandler(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download data: %w", err)
	}

	return nil
}

// uploadRequest uploads one block to the server over HTTP, counting it once the server confirms the whole body.
//
// Parameters:
//   - ctx: cancellation for the request.
//   - request: the server, its resolved upload URL, and the upload size.
//   - connection: the worker's connection, which counts the confirmed bytes.
//
// Returns:
//   - error: the request, status, or confirmation error.
func uploadRequest(
	ctx context.Context,
	request *transferRequest,
	connection *meter.Connection,
) error {
	size := ulSizes[request.size]
	chunkSize := int64(size*100-51) * 10
	body := &countingReader{reader: connection.NewChunk().UploadHandler(chunkSize)}

	target := request.uploadTarget()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, io.NopCloser(body))
	if err != nil {
		return fmt.Errorf("failed to create upload HTTP request: %w", err)
	}

	req.ContentLength = chunkSize
	request.client.log().Debug(ctx, "uploading", logKeyURL, target, logKeyBytes, req.ContentLength)
	req.Header.Set("Content-Type", "application/octet-stream")

	// Following a redirect would replace the POST with a bodyless GET, whose success confirms nothing. A redirect
	// is returned as is, and its status fails the check below.
	client := *request.client.doer
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform upload HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	err = checkStatus(resp)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read upload response: %w", err)
	}

	// A server can answer before reading the body, so the response confirms the upload only once the transport has
	// taken the whole body.
	sent := body.count.Load()
	if sent != chunkSize {
		return fmt.Errorf("%w: %d of %d bytes", errUploadIncomplete, sent, chunkSize)
	}

	connection.Confirm(chunkSize)

	return nil
}

// resolveUploadURL follows any redirects from a server's upload URL with a HEAD request. Some servers advertise an
// http URL that redirects to https and close the connection while a POST body is still streaming, so the upload
// fails before its redirect can be seen. Sending to the final URL avoids that.
//
// Parameters:
//   - ctx: cancellation for the HEAD request.
//   - server: the server.
//
// Returns:
//   - string: the URL the server's upload URL redirects to, or empty when the request fails or nothing redirects.
func (s *Client) resolveUploadURL(ctx context.Context, server *Server) string {
	ctx, cancel := context.WithTimeout(ctx, uploadResolveTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, server.URL, nil)
	if err != nil {
		return ""
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		s.log().Debug(ctx, "upload URL not resolved", logKeyError, err)

		return ""
	}

	_ = resp.Body.Close()

	if resp.Request == nil || resp.Request.URL == nil {
		return ""
	}

	resolved := resp.Request.URL.String()
	if resolved == server.URL {
		return ""
	}

	s.log().Debug(ctx, "resolved upload URL", logKeyURL, server.URL, logKeyResolved, resolved)

	return resolved
}

// resolveUploadURLs resolves the upload URL of every server, running at most limit resolutions at once, and returns
// when all are done.
//
// Parameters:
//   - ctx: cancellation for the HEAD requests.
//   - servers: the servers to resolve.
//   - limit: the most resolutions to run at once.
//
// Returns:
//   - []string: each server's resolved upload URL, in server order, or empty where nothing was resolved.
func (s *Client) resolveUploadURLs(ctx context.Context, servers Servers, limit int) []string {
	resolved := make([]string, len(servers))

	var resolving sync.WaitGroup

	slots := make(chan struct{}, limit)

	for index, server := range servers {
		slots <- struct{}{}

		resolving.Go(func() {
			defer func() { <-slots }()

			resolved[index] = s.resolveUploadURL(ctx, server)
		})
	}

	resolving.Wait()

	return resolved
}

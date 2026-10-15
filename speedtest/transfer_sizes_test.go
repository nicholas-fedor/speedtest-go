// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// TestCheckTransferSizes checks that 0 and every table value are accepted, and that other values are rejected
// with the accepted values listed.
func TestCheckTransferSizes(t *testing.T) {
	t.Parallel()

	require.NoError(t, CheckDownloadSize(0))
	require.NoError(t, CheckUploadSize(0))

	for _, size := range dlSizes {
		require.NoError(t, CheckDownloadSize(size), size)
	}

	for _, size := range ulSizes {
		require.NoError(t, CheckUploadSize(size), size)
	}

	err := CheckDownloadSize(800)
	require.ErrorIs(t, err, ErrInvalidTransferSize)
	assert.Contains(t, err.Error(), "800 is not a download size, use one of 350, 500, 750, 1000")

	err = CheckUploadSize(-1)
	require.ErrorIs(t, err, ErrInvalidTransferSize)
	assert.Contains(t, err.Error(), "use one of 100, 300, 500, 800")
}

// TestSizeIndex checks that a configured size maps to its table index, and that 0 or an unknown size falls back.
func TestSizeIndex(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, sizeIndex(dlSizes[:], 2000, defaultDownloadIndex))
	assert.Equal(t, 0, sizeIndex(ulSizes[:], 100, defaultUploadIndex))
	assert.Equal(t, defaultDownloadIndex, sizeIndex(dlSizes[:], 0, defaultDownloadIndex))
	assert.Equal(t, defaultUploadIndex, sizeIndex(ulSizes[:], 123, defaultUploadIndex))
}

// TestWithTransferSizes_Invalid checks that sizes outside the tables are rejected, and that both are reported.
func TestWithTransferSizes_Invalid(t *testing.T) {
	t.Parallel()

	client, err := New(WithTransferSizes(800, 2000))

	require.ErrorIs(t, err, ErrInvalidOption)
	require.ErrorIs(t, err, ErrInvalidTransferSize)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "download")
	assert.Contains(t, err.Error(), "upload")
}

// requestedPaths returns the paths the fake API recorded for an endpoint.
func requestedPaths(api *testserver.API, endpoint string) []string {
	var paths []string

	for _, req := range api.Requests() {
		if req.Endpoint == endpoint {
			paths = append(paths, req.Path)
		}
	}

	return paths
}

// uploadDeclaredSizes returns the body size each upload POST the fake API recorded declared. An upload states its
// full size before its body is sent, so the size is known even for an upload cut off when the test ends. The HEAD
// requests that resolve upload redirects share the upload path and are left out.
func uploadDeclaredSizes(api *testserver.API) []int64 {
	var sizes []int64

	for _, req := range api.Requests() {
		if req.Endpoint == testserver.PathUpload && req.Method == http.MethodPost {
			sizes = append(sizes, req.ContentLength)
		}
	}

	return sizes
}

// assertUploadSize checks that every upload declared the expected size. The fake records an upload once it stops
// reading the body, which for an upload cut off at the end of the test can be just after the test returns, so it
// first waits for at least one upload to be recorded.
func assertUploadSize(t *testing.T, api *testserver.API, want int64) {
	t.Helper()

	require.Eventually(t, func() bool { return len(uploadDeclaredSizes(api)) > 0 },
		5*time.Second, 10*time.Millisecond, "the test must send at least one upload")

	for _, size := range uploadDeclaredSizes(api) {
		assert.Equal(t, want, size)
	}
}

// TestHTTPTransferSizes checks that HTTP tests request the configured sizes, and the defaults without them,
// including the smaller default upload of multi-server tests.
func TestHTTPTransferSizes(t *testing.T) {
	t.Parallel()

	httpUpload := func(sizeKB int) int64 { return int64(sizeKB*100-51) * 10 }

	t.Run("configured", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := newTestClient(
			t,
			WithConnections(1),
			WithTransferSizes(2000, 300),
			WithTestDuration(200*time.Millisecond),
		)

		server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}

		require.NoError(t, server.DownloadTestContext(context.Background()))
		require.NoError(t, server.UploadTestContext(context.Background()))

		paths := requestedPaths(api, testserver.PathDownload)
		require.NotEmpty(t, paths)

		for _, path := range paths {
			assert.True(t, strings.HasSuffix(path, "/random2000x2000.jpg"), path)
		}

		assertUploadSize(t, api, httpUpload(300))
	})

	t.Run("multi-server defaults", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := newTestClient(t, WithConnections(1), WithTestDuration(200*time.Millisecond))

		pool := Servers{{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}}

		require.NoError(t, pool[0].MultiDownloadTestContext(context.Background(), pool))
		require.NoError(t, pool[0].MultiUploadTestContext(context.Background(), pool))

		for _, path := range requestedPaths(api, testserver.PathDownload) {
			assert.True(t, strings.HasSuffix(path, "/random1000x1000.jpg"), path)
		}

		assertUploadSize(t, api, httpUpload(800))
	})
}

// TestTCPTransferSizes checks that TCP tests request the configured sizes.
func TestTCPTransferSizes(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := newTestClient(
		t,
		WithProtocol(TCPTest),
		WithConnections(1),
		WithTransferSizes(2000, 300),
		WithTestDuration(200*time.Millisecond),
	)

	server := &Server{ID: "1001", Host: srv.Addr(), Context: client}

	require.NoError(t, server.DownloadTestContext(context.Background()))
	require.NoError(t, server.UploadTestContext(context.Background()))

	assert.Positive(t, countCommands(srv, "DOWNLOAD 2000000"))
	assert.Positive(t, countCommands(srv, "UPLOAD 300000 0"))
	assert.Zero(t, countPrefix(srv, "DOWNLOAD 1000000"))
}

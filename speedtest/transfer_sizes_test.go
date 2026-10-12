// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
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

// TestNewUserConfigInvalidSizes checks that sizes outside the tables are dropped in favor of the defaults.
func TestNewUserConfigInvalidSizes(t *testing.T) {
	t.Parallel()

	client := New(WithUserConfig(&UserConfig{DownloadSize: 800, UploadSize: 2000}))

	assert.Zero(t, client.config.DownloadSize)
	assert.Zero(t, client.config.UploadSize)
	assert.Equal(t, defaultDownloadIndex, client.downloadIndex(defaultDownloadIndex))
	assert.Equal(t, defaultUploadIndex, client.uploadIndex(defaultUploadIndex))
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

// uploadBodySizes returns the body sizes the fake API recorded for uploads.
func uploadBodySizes(api *testserver.API) []int64 {
	var sizes []int64

	for _, req := range api.Requests() {
		if req.Endpoint == testserver.PathUpload {
			sizes = append(sizes, req.BodySize)
		}
	}

	return sizes
}

// assertUploadSize checks that some upload sent a full body of the expected size and none sent more. Uploads still
// in flight when the test ends are cut off, so their bodies are shorter.
func assertUploadSize(t *testing.T, sizes []int64, want int64) {
	t.Helper()

	require.Contains(t, sizes, want)

	for _, size := range sizes {
		assert.LessOrEqual(t, size, want)
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
		client := New(
			WithUserConfig(&UserConfig{MaxConnections: 1, DownloadSize: 2000, UploadSize: 300}),
		)
		client.SetCaptureTime(200 * time.Millisecond)

		server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}

		require.NoError(t, server.DownloadTestContext(context.Background()))
		require.NoError(t, server.UploadTestContext(context.Background()))

		paths := requestedPaths(api, testserver.PathDownload)
		require.NotEmpty(t, paths)

		for _, path := range paths {
			assert.True(t, strings.HasSuffix(path, "/random2000x2000.jpg"), path)
		}

		assertUploadSize(t, uploadBodySizes(api), httpUpload(300))
	})

	t.Run("multi-server defaults", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := New(WithUserConfig(&UserConfig{MaxConnections: 1}))
		client.SetCaptureTime(200 * time.Millisecond)

		pool := Servers{{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}}

		require.NoError(t, pool[0].MultiDownloadTestContext(context.Background(), pool))
		require.NoError(t, pool[0].MultiUploadTestContext(context.Background(), pool))

		for _, path := range requestedPaths(api, testserver.PathDownload) {
			assert.True(t, strings.HasSuffix(path, "/random1000x1000.jpg"), path)
		}

		assertUploadSize(t, uploadBodySizes(api), httpUpload(800))
	})
}

// TestTCPTransferSizes checks that TCP tests request the configured sizes.
func TestTCPTransferSizes(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)
	client := New(WithUserConfig(&UserConfig{
		TestMode:       TCPTest,
		MaxConnections: 1,
		DownloadSize:   2000,
		UploadSize:     300,
	}))
	client.SetCaptureTime(200 * time.Millisecond)

	server := &Server{ID: "1001", Host: srv.Addr(), Context: client}

	require.NoError(t, server.DownloadTestContext(context.Background()))
	require.NoError(t, server.UploadTestContext(context.Background()))

	assert.Positive(t, countCommands(srv, "DOWNLOAD 2000000"))
	assert.Positive(t, countCommands(srv, "UPLOAD 300000 0"))
	assert.Zero(t, countPrefix(srv, "DOWNLOAD 1000000"))
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingReader returns some data and then an error, like a connection the server drops.
type failingReader struct{}

// Errors of the test readers.
var (
	// errBrokenBody is the error of a response body that fails mid-read.
	errBrokenBody = errors.New("broken body")
)

// Read returns 10 bytes and errBrokenBody.
//
// Parameters:
//   - buffer: the destination.
//
// Returns:
//   - int: 10.
//   - error: errBrokenBody.
func (failingReader) Read(buffer []byte) (int, error) {
	return copy(buffer, make([]byte, 10)), errBrokenBody
}

// TestChunk_DownloadHandler checks that a download counts every byte of a body toward its connection, returns read
// errors other than EOF, and refuses a second body.
func TestChunk_DownloadHandler(t *testing.T) {
	t.Parallel()

	connection := newTestMeter(1, 0).DownloadConnection()
	chunk := connection.NewChunk()

	require.NoError(t, chunk.DownloadHandler(bytes.NewReader(make([]byte, 20_000))))
	assert.Equal(t, int64(20_000), connection.Bytes())

	require.ErrorIs(t, chunk.DownloadHandler(bytes.NewReader([]byte("again"))), errChunkReused)

	require.ErrorIs(t, connection.NewChunk().DownloadHandler(failingReader{}), errBrokenBody)
	assert.Equal(t, int64(20_010), connection.Bytes(), "bytes before the error still count")
}

// TestChunk_DownloadHandlerStopped checks that a download ends without reading once its test has stopped.
func TestChunk_DownloadHandlerStopped(t *testing.T) {
	t.Parallel()

	connection := newTestMeter(1, 0).DownloadConnection()
	connection.direction.stopped.Store(true)

	require.NoError(t, connection.NewChunk().DownloadHandler(zeroReader{}))
	assert.Zero(t, connection.Bytes())
}

// TestChunk_Read checks that an upload chunk supplies exactly its size in reads of at most readChunkSize, filled
// with the upload pattern.
func TestChunk_Read(t *testing.T) {
	t.Parallel()

	chunk := newTestMeter(1, 0).UploadConnection().NewChunk().UploadHandler(readChunkSize + 100)
	buffer := make([]byte, 4*readChunkSize)

	read, err := chunk.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, readChunkSize, read)
	assert.Equal(t, bytes.Repeat([]byte{uploadPatternByte}, readChunkSize), buffer[:read])

	read, err = chunk.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, 100, read)

	read, err = chunk.Read(buffer)
	require.ErrorIs(t, err, io.EOF)
	assert.Zero(t, read)
}

// TestChunk_UploadCounts checks that reading an upload chunk adds to the direction's sent volume but not to the
// connection's bytes, which count only what the server confirms. The transport reads ahead into buffers, so bytes
// are counted only once the server confirms the request.
func TestChunk_UploadCounts(t *testing.T) {
	t.Parallel()

	connection := newTestMeter(1, 0).UploadConnection()

	read, err := io.Copy(io.Discard, connection.NewChunk().UploadHandler(3*readChunkSize))
	require.NoError(t, err)

	assert.Equal(t, int64(3*readChunkSize), read)
	assert.Equal(t, int64(3*readChunkSize), connection.direction.read.Load())
	assert.Zero(t, connection.Bytes())

	connection.Confirm(3 * readChunkSize)
	assert.Equal(t, int64(3*readChunkSize), connection.Bytes())
}

// TestChunk_UploadHandlerSize checks that an upload body needs a positive size.
func TestChunk_UploadHandlerSize(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { newTestMeter(1, 0).UploadConnection().NewChunk().UploadHandler(0) })
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// Chunk counts the data of one download or upload request toward its meter.
type Chunk struct {
	// meter is the meter the bytes count toward.
	meter *Meter
	// uploadRead counts the bytes read from an upload chunk into its direction's read volume.
	uploadRead *atomic.Int64
	// remaining is the upload bytes left to read.
	remaining int64
	// used reports that the chunk has started handling a request.
	used bool
}

// Chunk sizes.
const (
	// readChunkSize is how many bytes an upload chunk hands out per read, small enough for frequent rate feedback.
	readChunkSize = 1024
	// discardBufferSize is the size of the buffers download chunks read into and discard.
	discardBufferSize = 8192
)

// Errors of chunk handlers.
var (
	// errChunkReused is returned when a chunk that already handled a request is used again.
	errChunkReused = errors.New("a chunk handles only one request")
	// errDiscardBuffer is returned when the discard buffer pool hands out something other than a buffer.
	errDiscardBuffer = errors.New("failed to get a discard buffer")
)

// discardBuffers holds the buffers download chunks read into and discard.
var discardBuffers = sync.Pool{
	New: func() any {
		buffer := make([]byte, discardBufferSize)

		return &buffer
	},
}

// DownloadHandler reads a download response body until it ends or the download test stops, counting each byte
// read.
//
// Parameters:
//   - reader: the response body.
//
// Returns:
//   - error: the read error other than [io.EOF], or an error when the chunk was already used.
func (c *Chunk) DownloadHandler(reader io.Reader) error {
	if c.used {
		return errChunkReused
	}

	c.used = true

	buffer, ok := discardBuffers.Get().(*[]byte)
	if !ok {
		return errDiscardBuffer
	}
	defer discardBuffers.Put(buffer)

	download := c.meter.currentDownload()

	for !download.stopped.Load() {
		read, err := reader.Read(*buffer)
		download.confirmed.Add(int64(read))

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err //nolint:wrapcheck // The request functions wrap transfer errors with their context.
		}
	}

	return nil
}

// UploadHandler prepares the chunk to supply an upload body of the given size.
//
// Bytes read from the chunk add to the upload's read volume. They count as uploaded only once the request's caller
// confirms them with [Meter.AddTotalUpload].
//
// Parameters:
//   - size: the body size in bytes, greater than 0.
//
// Returns:
//   - *Chunk: the chunk, which reads as the body.
func (c *Chunk) UploadHandler(size int64) *Chunk {
	if size <= 0 {
		panic("meter: an upload body needs a positive size")
	}

	c.used = true
	c.remaining = size
	c.uploadRead = &c.meter.currentUpload().read

	return c
}

// Read fills buffer with upload data until the body's size has been read.
//
// Parameters:
//   - buffer: the destination.
//
// Returns:
//   - int: the bytes read.
//   - error: [io.EOF] once the whole body has been read.
func (c *Chunk) Read(buffer []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, io.EOF
	}

	read := copy(buffer, c.meter.pattern[:min(c.remaining, readChunkSize)])
	c.remaining -= int64(read)

	if c.uploadRead != nil {
		c.uploadRead.Add(int64(read))
	}

	return read, nil
}

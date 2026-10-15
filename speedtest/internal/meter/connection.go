// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import (
	"sync"
	"sync/atomic"
	"time"
)

// Handler sends one request of a test, counting its data through the worker's connection.
//
// Parameters:
//   - connection: the connection of the worker that runs the handler.
//
// Returns:
//   - error: the request's error, which counts as a failed request.
type Handler func(connection *Connection) error

// Connection is one worker of a test. It counts the bytes, requests, and failures of the requests the worker sends,
// and the time it spends in them.
type Connection struct {
	// direction is the test the connection belongs to.
	direction *Direction
	// calls are the intervals the worker spent in requests. mu guards them.
	calls []interval
	// bytes counts the bytes downloaded, or the upload bytes the server confirmed.
	bytes atomic.Int64
	// sent counts the upload bytes read into the transport, confirmed or not.
	sent atomic.Int64
	// handler is the index of the handler the worker runs, or noHandler outside a test run.
	handler int
	// requests counts the requests sent. mu guards it.
	requests int
	// failures counts the requests that returned an error. mu guards it.
	failures int
	// mu guards calls, requests, and failures.
	mu sync.Mutex
}

// interval is a span of time from start to end.
type interval struct {
	// start is when the span began.
	start time.Time
	// end is when the span ended.
	end time.Time
}

// noHandler marks a connection that runs no registered handler.
const noHandler = -1

// newConnection returns a connection with no counts.
//
// Parameters:
//   - direction: the test it belongs to.
//   - handler: the index of the handler its worker runs, or noHandler.
//
// Returns:
//   - *Connection: the connection.
func newConnection(direction *Direction, handler int) *Connection {
	return &Connection{direction: direction, handler: handler}
}

// NewChunk returns a chunk that counts one request's data toward this connection.
//
// Returns:
//   - *Chunk: the chunk, ready for [Chunk.DownloadHandler] or [Chunk.UploadHandler].
func (c *Connection) NewChunk() *Chunk {
	return &Chunk{connection: c}
}

// Confirm counts upload bytes that the server confirmed.
//
// Parameters:
//   - bytes: the confirmed bytes.
func (c *Connection) Confirm(bytes int64) {
	c.bytes.Add(bytes)
}

// Bytes returns the bytes downloaded, or the upload bytes the server confirmed, so far.
//
// Returns:
//   - int64: the bytes.
func (c *Connection) Bytes() int64 {
	return c.bytes.Load()
}

// Sent returns the upload bytes read into the transport so far, confirmed or not.
//
// Returns:
//   - int64: the bytes, 0 for downloads.
func (c *Connection) Sent() int64 {
	return c.sent.Load()
}

// run sends one request with the handler and records it.
//
// Parameters:
//   - handler: the handler.
func (c *Connection) run(handler Handler) {
	start := time.Now()
	err := handler(c)
	end := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls = append(c.calls, interval{start: start, end: end})
	c.requests++

	if err != nil {
		c.failures++
	}
}

// result returns the connection's counts, with its time in requests cut off when the test stopped.
//
// Parameters:
//   - bytes: the connection's bytes when the test stopped.
//   - stoppedAt: when the test stopped.
//
// Returns:
//   - ConnectionResult: the counts.
func (c *Connection) result(bytes int64, stoppedAt time.Time) ConnectionResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	var active time.Duration

	for _, call := range c.calls {
		if !call.start.Before(stoppedAt) {
			continue
		}

		end := call.end
		if end.After(stoppedAt) {
			end = stoppedAt
		}

		active += end.Sub(call.start)
	}

	return ConnectionResult{
		Handler:  c.handler,
		Bytes:    bytes,
		Requests: c.requests,
		Failures: c.failures,
		Active:   active,
	}
}

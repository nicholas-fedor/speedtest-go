// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import "time"

// Result is the measurement of one test, taken when it stopped.
type Result struct {
	// Connections are the counts of each worker, in worker order.
	Connections []ConnectionResult
	// Rate is the bytes per second over the capture window.
	Rate float64
	// Bytes is the bytes downloaded, or the upload bytes the server confirmed, by the time the test stopped. It is
	// the sum of the connections' bytes.
	Bytes int64
	// Sent is the upload bytes read into the transport by the time the test stopped, confirmed or not. It is 0 for
	// downloads.
	Sent int64
	// Window is the capture window, from the start of the test until it stopped.
	Window time.Duration
	// Requests is the number of requests the workers sent, including those cut short by the stop.
	Requests int
	// Failures is the number of requests that returned an error.
	Failures int
}

// ConnectionResult is the measurement of one worker of a test.
type ConnectionResult struct {
	// Handler is the index of the handler the worker ran, which names its server in a multi-server test.
	Handler int
	// Bytes is the bytes the worker downloaded, or the upload bytes the server confirmed, by the time the test
	// stopped.
	Bytes int64
	// Requests is the number of requests the worker sent.
	Requests int
	// Failures is the number of the worker's requests that returned an error.
	Failures int
	// Active is the time the worker spent in requests before the test stopped. Upload workers can wait idle while
	// the adaptive worker count is lower.
	Active time.Duration
}

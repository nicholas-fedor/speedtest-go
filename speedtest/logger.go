// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import "context"

// Logger receives a client's debug messages.
//
// Messages are static strings, and args alternate keys and values in the style of [log/slog], so a [*slog.Logger]
// fits behind a one-line adapter that calls its DebugContext method. A client without a logger discards its messages.
type Logger interface {
	// Debug records a debug message.
	//
	// Parameters:
	//   - ctx: the context of the operation that logs the message.
	//   - msg: a static message.
	//   - args: alternating keys and values.
	Debug(ctx context.Context, msg string, args ...any)
}

// Log keys of the client's debug messages.
const (
	// logKeyAddress is a host or address that is pinged.
	logKeyAddress = "address"
	// logKeyArch is the architecture the program runs on.
	logKeyArch = "arch"
	// logKeyBacklog is the volume of uploaded data not yet confirmed, in bytes.
	logKeyBacklog = "backlog"
	// logKeyBaseURL is the speedtest API base URL.
	logKeyBaseURL = "baseURL"
	// logKeyBytes is a request body size in bytes.
	logKeyBytes = "bytes"
	// logKeyConnections is the number of connections a test uses.
	logKeyConnections = "connections"
	// logKeyCount is a number of items, such as servers.
	logKeyCount = "count"
	// logKeyCPUs is the number of CPUs.
	logKeyCPUs = "cpus"
	// logKeyDirection is a transfer direction, "download" or "upload".
	logKeyDirection = "direction"
	// logKeyError is an error that the client recovered from.
	logKeyError = "error"
	// logKeyHandlers is the number of transfer handlers registered for a test.
	logKeyHandlers = "handlers"
	// logKeyLatency is a measured round-trip time.
	logKeyLatency = "latency"
	// logKeyLatencies is a list of measured round-trip times.
	logKeyLatencies = "latencies"
	// logKeyLimit is the most workers a test may run.
	logKeyLimit = "limit"
	// logKeyMainWorkers is the number of workers given to the main transfer handler.
	logKeyMainWorkers = "mainWorkers"
	// logKeyOS is the operating system the program runs on.
	logKeyOS = "os"
	// logKeyOtherWorkers is the number of workers shared by the other transfer handlers.
	logKeyOtherWorkers = "otherWorkers"
	// logKeyPingMode is the method used to ping servers.
	logKeyPingMode = "pingMode"
	// logKeyProtocol is the protocol used for download and upload tests.
	logKeyProtocol = "protocol"
	// logKeyProxy is the HTTP proxy, with any password redacted.
	logKeyProxy = "proxy"
	// logKeyRate is a confirmed transfer rate in bytes per second.
	logKeyRate = "rate"
	// logKeyResolved is the URL that a request was redirected to.
	logKeyResolved = "resolved"
	// logKeySearch is the keyword that filters the server list.
	logKeySearch = "search"
	// logKeySource is where the client's connections leave from.
	logKeySource = "source"
	// logKeyURL is the URL of a request.
	logKeyURL = "url"
	// logKeyWorkers is the number of workers currently running.
	logKeyWorkers = "workers"
)

// noopLogger discards every message. Clients without a [Logger] use it.
type noopLogger struct{}

var _ Logger = noopLogger{}

// Debug discards the message.
func (noopLogger) Debug(context.Context, string, ...any) {}

// loggerOrNoop returns a logger that is safe to call.
//
// Parameters:
//   - logger: the configured logger, or nil.
//
// Returns:
//   - Logger: logger, or a [noopLogger] when it is nil.
func loggerOrNoop(logger Logger) Logger {
	if logger == nil {
		return noopLogger{}
	}

	return logger
}

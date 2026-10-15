// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package meter

import "context"

// Logger receives the meter's debug messages. It matches the speedtest client's logger.
type Logger interface {
	// Debug records a debug message.
	//
	// Parameters:
	//   - ctx: the context of the test that logs the message.
	//   - msg: a static message.
	//   - args: alternating keys and values.
	Debug(ctx context.Context, msg string, args ...any)
}

// noopLogger discards every message. Meters without a [Logger] use it.
type noopLogger struct{}

// Log keys of the meter's debug messages.
const (
	// logKeyBacklog is the volume of uploaded data not yet confirmed, in bytes.
	logKeyBacklog = "backlog"
	// logKeyHandlers is the number of transfer handlers registered for a test.
	logKeyHandlers = "handlers"
	// logKeyLimit is the most workers a test may run.
	logKeyLimit = "limit"
	// logKeyMainWorkers is the number of workers given to the main transfer handler.
	logKeyMainWorkers = "mainWorkers"
	// logKeyOtherWorkers is the number of workers shared by the other transfer handlers.
	logKeyOtherWorkers = "otherWorkers"
	// logKeyRate is a confirmed transfer rate in bytes per second.
	logKeyRate = "rate"
	// logKeyWorkers is the number of workers currently running.
	logKeyWorkers = "workers"
)

var _ Logger = noopLogger{}

// Debug discards the message.
func (noopLogger) Debug(context.Context, string, ...any) {}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"io"
	"log/slog"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// debugLogger writes the speedtest client's debug messages through [slog], for --debug.
type debugLogger struct {
	// logger writes the messages.
	logger *slog.Logger
}

var _ speedtest.Logger = debugLogger{}

// newDebugLogger returns a logger that writes debug messages as text.
//
// Parameters:
//   - writer: where the messages go, standard error for the CLI, so they never mix into results on standard output.
//
// Returns:
//   - speedtest.Logger: the logger.
func newDebugLogger(writer io.Writer) speedtest.Logger {
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})

	return debugLogger{logger: slog.New(handler)}
}

// Debug writes a debug message.
//
// Parameters:
//   - ctx: the context of the operation that logs the message.
//   - msg: the message.
//   - args: alternating keys and values.
func (l debugLogger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

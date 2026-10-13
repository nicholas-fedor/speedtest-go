// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoggerOrNoop checks that a nil logger is replaced by one that discards messages, and a set one is kept.
func TestLoggerOrNoop(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}

	assert.IsType(t, noopLogger{}, loggerOrNoop(nil))
	assert.Same(t, logger, loggerOrNoop(logger))
}

// TestLog_ZeroValues checks that clients, servers, and data managers built without New log nowhere instead of
// panicking.
func TestLog_ZeroValues(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		(*Client)(nil).log().Debug(context.Background(), "message")
		(&Client{}).log().Debug(context.Background(), "message")
		(*Server)(nil).log().Debug(context.Background(), "message")
		(&Server{}).log().Debug(context.Background(), "message")
		(*DataManager)(nil).log().Debug(context.Background(), "message")
		NewDataManager().log().Debug(context.Background(), "message")
	})
}

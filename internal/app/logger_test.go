// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDebugLogger checks that debug messages are written as text with their keys and values.
func TestDebugLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	newDebugLogger(
		&buf,
	).Debug(context.Background(), "retrieving user info", "url", "http://example.test")

	assert.Contains(
		t,
		buf.String(),
		`level=DEBUG msg="retrieving user info" url=http://example.test`,
	)
}

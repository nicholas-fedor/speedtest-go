// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package version

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCommand checks that the command prints its version line and takes no arguments.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	command := NewCommand("speedtest-go v1.2.3", &out)
	command.SetArgs([]string{})

	require.NoError(t, command.ExecuteContext(t.Context()))
	assert.Equal(t, "speedtest-go v1.2.3\n", out.String())

	require.Error(t, command.Args(command, []string{"extra"}))
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package cities

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/cities/mocks"
)

// TestNewCommand checks that the command writes the city labels to its output and returns the writer's error.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	errWrite := errors.New("broken pipe")

	var out bytes.Buffer

	writer := mocks.NewMockWriter(t)
	writer.EXPECT().WriteCities(&out).Return(errWrite).Once()

	command := NewCommand(writer, &out)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{})

	require.ErrorIs(t, command.ExecuteContext(t.Context()), errWrite)
}

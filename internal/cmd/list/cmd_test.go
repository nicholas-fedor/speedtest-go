// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package list

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/list/mocks"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// errSettings stands in for settings that fail to resolve.
var errSettings = errors.New("invalid settings")

// TestNewCommand checks that the command passes the resolved settings and its output to the lister, and returns the
// lister's error.
func TestNewCommand(t *testing.T) {
	t.Parallel()

	errLookup := errors.New("lookup failed")
	cfg := config.Config{Search: "tokyo"}

	var out bytes.Buffer

	lister := mocks.NewMockLister(t)
	lister.EXPECT().List(mock.Anything, cfg, &out).Return(errLookup).Once()

	command := NewCommand(
		lister,
		func(*pflag.FlagSet) (config.Config, error) { return cfg, nil },
		&out,
	)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{})

	require.ErrorIs(t, command.ExecuteContext(t.Context()), errLookup)
}

// TestNewCommandSettingsError checks that settings that fail to resolve stop the command before any lookup.
func TestNewCommandSettingsError(t *testing.T) {
	t.Parallel()

	command := NewCommand(mocks.NewMockLister(t), func(*pflag.FlagSet) (config.Config, error) {
		return config.Config{}, errSettings
	}, &bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{})

	assert.ErrorIs(t, command.ExecuteContext(t.Context()), errSettings)
}

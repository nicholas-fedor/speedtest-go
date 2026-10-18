// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterRootFlagsAddsPersistentFlags checks the flags every command takes.
func TestRegisterRootFlagsAddsPersistentFlags(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	RegisterRootFlags(rootCmd)

	pf := rootCmd.PersistentFlags()
	require.NotNil(t, pf.Lookup("config"))
	require.NotNil(t, pf.Lookup("proxy"))
	require.NotNil(t, pf.Lookup("source"))
	require.NotNil(t, pf.Lookup("dns-bind-source"))
	require.NotNil(t, pf.Lookup("ua"))
	require.NotNil(t, pf.Lookup("debug"))
	require.NotNil(t, pf.Lookup("base-url"))
	require.NotNil(t, pf.Lookup("dscp"))
	require.NotNil(t, pf.Lookup("filter-cc"))
	require.NotNil(t, pf.Lookup("mask-isp"), "list honors --mask-isp too")
}

// TestRegisterRootFlagsAddsLocalFlags checks the flags only the speed test takes.
func TestRegisterRootFlagsAddsLocalFlags(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	RegisterRootFlags(rootCmd)

	f := rootCmd.Flags()
	require.NotNil(t, f.Lookup("server"))
	require.NotNil(t, f.Lookup("custom-url"))
	require.NotNil(t, f.Lookup("saving-mode"))
	require.NotNil(t, f.Lookup("json"))
	require.NotNil(t, f.Lookup("jsonl"))
	require.NotNil(t, f.Lookup("unix"))
	require.NotNil(t, f.Lookup("multi"))
	require.NotNil(t, f.Lookup("thread"))
	require.NotNil(t, f.Lookup("no-download"))
	require.NotNil(t, f.Lookup("no-upload"))
	require.NotNil(t, f.Lookup("ping-mode"))
	require.NotNil(t, f.Lookup("protocol"))
	require.NotNil(t, f.Lookup("dl-size"))
	require.NotNil(t, f.Lookup("ul-size"))
	require.NotNil(t, f.Lookup("unit"))
}

// TestRegisterRootFlagsLocation checks that the location flags are persistent, so the speed test and list both
// accept them.
func TestRegisterRootFlagsLocation(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	RegisterRootFlags(rootCmd)

	persistent := rootCmd.PersistentFlags()
	require.NoError(t, persistent.Set(Location, "60,-110"))
	require.NoError(t, persistent.Set(City, "capetown"))
	require.NoError(t, persistent.Set(Search, "fiber"))

	assert.Equal(t, "60,-110", persistent.Lookup(Location).Value.String())
	assert.Equal(t, "capetown", persistent.Lookup(City).Value.String())
	assert.Equal(t, "fiber", persistent.Lookup(Search).Value.String())
}

// TestRegisterRootFlagsDefaults checks the defaults that are not zero values.
func TestRegisterRootFlagsDefaults(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	RegisterRootFlags(rootCmd)

	assert.Equal(t, "http", rootCmd.Flags().Lookup(PingMode).DefValue)
	assert.Equal(t, "http", rootCmd.Flags().Lookup(Protocol).DefValue)
}

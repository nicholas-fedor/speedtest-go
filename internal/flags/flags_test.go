package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRootFlagsAddsPersistentFlags(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	var cfgFile string

	RegisterRootFlags(rootCmd, &cfgFile)

	pf := rootCmd.PersistentFlags()
	require.NotNil(t, pf.Lookup("config"))
	require.NotNil(t, pf.Lookup("proxy"))
	require.NotNil(t, pf.Lookup("source"))
	require.NotNil(t, pf.Lookup("dns-bind-source"))
	require.NotNil(t, pf.Lookup("ua"))
	require.NotNil(t, pf.Lookup("debug"))
}

func TestRegisterRootFlagsAddsLocalFlags(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	var cfgFile string

	RegisterRootFlags(rootCmd, &cfgFile)

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
	require.NotNil(t, f.Lookup("unit"))
}

func TestRegisterRootFlagsBindsViper(t *testing.T) {
	t.Parallel()

	rootCmd := &cobra.Command{Use: "test"}

	var cfgFile string

	RegisterRootFlags(rootCmd, &cfgFile)

	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("proxy"))
	assert.NotNil(t, rootCmd.Flags().Lookup("json"))
}

func TestRegisterListFlagsAddsFlags(t *testing.T) {
	t.Parallel()

	listCmd := &cobra.Command{Use: "list"}

	RegisterListFlags(listCmd)

	f := listCmd.Flags()
	require.NotNil(t, f.Lookup("location"))
	require.NotNil(t, f.Lookup("city"))
	require.NotNil(t, f.Lookup("search"))
}

func TestRegisterListFlagsBindsViper(t *testing.T) {
	t.Parallel()

	listCmd := &cobra.Command{Use: "list"}

	RegisterListFlags(listCmd)

	assert.NotNil(t, listCmd.Flags().Lookup("location"))
}

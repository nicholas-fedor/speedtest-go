package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // mutates global viper
func TestRegisterRootFlagsAddsPersistentFlags(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

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
	require.NotNil(t, pf.Lookup("base-url"))
	require.NotNil(t, pf.Lookup("dscp"))
	require.NotNil(t, pf.Lookup("filter-cc"))
}

//nolint:paralleltest // mutates global viper
func TestRegisterRootFlagsAddsLocalFlags(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

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
	require.NotNil(t, f.Lookup("protocol"))
	require.NotNil(t, f.Lookup("dl-size"))
	require.NotNil(t, f.Lookup("ul-size"))
	require.NotNil(t, f.Lookup("unit"))
}

//nolint:paralleltest // mutates global viper
func TestRegisterRootFlagsBindsViper(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	rootCmd := &cobra.Command{Use: "test"}

	var cfgFile string

	RegisterRootFlags(rootCmd, &cfgFile)

	require.NoError(t, rootCmd.PersistentFlags().Set("proxy", "socks5://127.0.0.1:1080"))
	require.NoError(t, rootCmd.Flags().Set("json", "true"))
	require.NoError(t, rootCmd.Flags().Set("thread", "8"))
	require.NoError(t, rootCmd.Flags().Set("ping-mode", "tcp"))
	require.NoError(t, rootCmd.PersistentFlags().Set("base-url", "http://127.0.0.1:8080/mirror"))

	assert.Equal(t, "socks5://127.0.0.1:1080", viper.GetString("proxy"))
	assert.True(t, viper.GetBool("json"))
	assert.Equal(t, 8, viper.GetInt("thread"))
	assert.Equal(t, "tcp", viper.GetString("ping-mode"))
	assert.Equal(t, "http://127.0.0.1:8080/mirror", viper.GetString("base-url"))
}

// TestRegisterRootFlagsLocation checks that the location flags are persistent, so the speed test and list both
// accept them, and that they are bound to viper.
//
//nolint:paralleltest // mutates global viper
func TestRegisterRootFlagsLocation(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	rootCmd := &cobra.Command{Use: "test"}

	var cfgFile string

	RegisterRootFlags(rootCmd, &cfgFile)

	persistent := rootCmd.PersistentFlags()
	require.NoError(t, persistent.Set("location", "60,-110"))
	require.NoError(t, persistent.Set("city", "capetown"))
	require.NoError(t, persistent.Set("search", "fiber"))

	assert.Equal(t, "60,-110", viper.GetString("location"))
	assert.Equal(t, "capetown", viper.GetString("city"))
	assert.Equal(t, "fiber", viper.GetString("search"))
}

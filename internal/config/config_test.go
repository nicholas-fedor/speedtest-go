package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupSavingModeEnablesUnixOutput(
	t *testing.T,
) { //nolint:paralleltest // Setup mutates global logger
	cfg := Config{
		SavingMode:  true,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.True(
		t,
		result.UnixOutput,
		"Setup should enable UnixOutput when SavingMode is active and no structured output is selected",
	)
}

func TestSetupDoesNotOverrideExplicitUnixOutput(
	t *testing.T,
) { //nolint:paralleltest // Setup mutates global logger
	cfg := Config{
		SavingMode: true,
		JSONOutput: true,
		UnixOutput: false,
	}

	result := Setup(cfg)

	assert.False(t, result.UnixOutput, "Setup should not override explicit JSONOutput")
}

func TestSetupDoesNotMutateWhenSavingModeInactive(t *testing.T) {
	t.Parallel()

	cfg := Config{
		SavingMode:  false,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.False(
		t,
		result.UnixOutput,
		"Setup should not modify UnixOutput when SavingMode is inactive",
	)
	assert.False(t, cfg.UnixOutput, "Setup should not mutate the caller's Config value")
}

func TestLoadReadsViperState(t *testing.T) { //nolint:paralleltest // mutates global viper
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("server", []int{1, 2})
	viper.Set("custom-url", "http://example.com")
	viper.Set("saving-mode", true)
	viper.Set("json", true)
	viper.Set("jsonl", false)
	viper.Set("unix", false)
	viper.Set("location", "40.7128,-74.0060")
	viper.Set("city", "New York")
	viper.Set("proxy", "http://proxy:8080")
	viper.Set("source", "eth0")
	viper.Set("dns-bind-source", true)
	viper.Set("multi", false)
	viper.Set("thread", 4)
	viper.Set("search", "keyword")
	viper.Set("ua", "test-agent")
	viper.Set("no-download", true)
	viper.Set("no-upload", false)
	viper.Set("ping-mode", "icmp")
	viper.Set("unit", "binary-bytes")
	viper.Set("debug", true)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2}, cfg.ServerIDs)
	assert.Equal(t, "http://example.com", cfg.CustomURL)
	assert.True(t, cfg.SavingMode)
	assert.True(t, cfg.JSONOutput)
	assert.False(t, cfg.JSONLOutput)
	assert.False(t, cfg.UnixOutput)
	assert.Equal(t, "40.7128,-74.0060", cfg.Location)
	assert.Equal(t, "New York", cfg.City)
	assert.Equal(t, "http://proxy:8080", cfg.Proxy)
	assert.Equal(t, "eth0", cfg.Source)
	assert.True(t, cfg.DNSBindSource)
	assert.False(t, cfg.Multi)
	assert.Equal(t, 4, cfg.Thread)
	assert.Equal(t, "keyword", cfg.Search)
	assert.Equal(t, "test-agent", cfg.UserAgent)
	assert.True(t, cfg.NoDownload)
	assert.False(t, cfg.NoUpload)
	assert.Equal(t, "icmp", cfg.PingMode)
	assert.Equal(t, "binary-bytes", cfg.Unit)
	assert.True(t, cfg.Debug)
}

func TestLoadReturnsDefaultsWhenViperEmpty(
	t *testing.T,
) { //nolint:paralleltest // mutates global viper
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Empty(t, cfg.ServerIDs)
	assert.Empty(t, cfg.CustomURL)
	assert.False(t, cfg.SavingMode)
	assert.False(t, cfg.JSONOutput)
	assert.False(t, cfg.JSONLOutput)
	assert.False(t, cfg.UnixOutput)
	assert.Empty(t, cfg.Location)
	assert.Empty(t, cfg.City)
	assert.Empty(t, cfg.Proxy)
	assert.Empty(t, cfg.Source)
	assert.False(t, cfg.DNSBindSource)
	assert.False(t, cfg.Multi)
	assert.Equal(t, 0, cfg.Thread)
	assert.Empty(t, cfg.Search)
	assert.Empty(t, cfg.UserAgent)
	assert.False(t, cfg.NoDownload)
	assert.False(t, cfg.NoUpload)
	assert.Empty(t, cfg.PingMode)
	assert.Empty(t, cfg.Unit)
	assert.False(t, cfg.Debug)
}

func TestInitViperWithConfigFile(t *testing.T) { //nolint:paralleltest // mutates global viper
	tmpFile, err := os.CreateTemp(t.TempDir(), "speedtest-go*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString("json: true\n")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	viper.Reset()
	t.Cleanup(viper.Reset)

	err = InitViper(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, tmpFile.Name(), viper.ConfigFileUsed())
	assert.True(t, viper.GetBool("json"))
}

func TestInitViperWithDefaultPath(t *testing.T) { //nolint:paralleltest // mutates global viper
	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper("")
	require.NoError(t, err)

	// When no config file exists, ConfigFileUsed returns empty string.
	assert.Empty(t, viper.ConfigFileUsed())
}

func TestInitViperPropagatesParseError(t *testing.T) { //nolint:paralleltest // mutates global viper
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("json: [\n"), 0o600))

	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestInitViperPropagatesMissingExplicitFile(
	t *testing.T,
) { //nolint:paralleltest // mutates global viper
	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestSetupDoesNotSuppressLogInPlainMode(
	t *testing.T,
) { //nolint:paralleltest // mutates global logger
	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("separator")

	assert.Contains(t, buf.String(), "separator",
		"Setup should not suppress the standard logger in plain human-readable mode",
	)
}

func TestSetupSuppressesLogInJSONMode(t *testing.T) { //nolint:paralleltest // mutates global logger
	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{
		JSONOutput:  true,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when JSONOutput is enabled",
	)
}

func TestSetupSuppressesLogInJSONLMode(
	t *testing.T,
) { //nolint:paralleltest // mutates global logger
	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: true,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when JSONLOutput is enabled",
	)
}

func TestSetupSuppressesLogInUnixMode(t *testing.T) { //nolint:paralleltest // mutates global logger
	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  true,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when UnixOutput is enabled",
	)
}

func TestSetupSuppressesLogWhenSavingModeEnablesUnixOutput(
	t *testing.T,
) { //nolint:paralleltest // mutates global logger
	var buf bytes.Buffer

	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{
		SavingMode:  true,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.True(t, result.UnixOutput,
		"Setup should enable UnixOutput when SavingMode is active",
	)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when SavingMode converts to UnixOutput",
	)
}

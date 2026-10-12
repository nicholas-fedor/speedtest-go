package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// initOnce registers the commands and flags once per test binary, since repeated runs reuse the global root command.
var initOnce sync.Once

// TestExecuteFlagErrorIsInvalidSetting checks that a flag value cobra cannot parse is reported as an invalid
// setting, so the CLI exits with code 2 for it. Parsing fails before any initializer reads the settings file.
//
//nolint:paralleltest // Init registers flags on the package-level root command.
func TestExecuteFlagErrorIsInvalidSetting(t *testing.T) {
	out, err := execute(t, "--thread", "abc")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.Contains(t, out, `invalid argument "abc"`)
	assert.Contains(t, out, "Usage:")
}

// execute runs the root command with args and returns its error and combined output. It always passes an
// unreachable --base-url, so no case can reach speedtest.net.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	initOnce.Do(Init)

	// The pre-run hook silences usage on the shared command, and the CLI only ever executes once.
	rootCmd.SilenceUsage = false

	var out bytes.Buffer

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append(args, "--base-url", "http://127.0.0.1:9"))

	err := Execute()

	return out.String(), err
}

// TestExecuteSettingsFileErrorIsInvalidSetting checks that a settings file that cannot be read is an invalid
// setting, so the CLI exits with code 2, and that the error is not followed by the usage text.
//
//nolint:paralleltest // Init registers flags on the package-level root command.
func TestExecuteSettingsFileErrorIsInvalidSetting(t *testing.T) {
	out, err := execute(t, "--config", filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, "failed to read config file")
	assert.NotContains(t, out, "Usage:")
}

// TestExecuteRunErrorSkipsUsage checks that an error after flag parsing, here an unknown ping mode, is reported
// without the usage text, which only helps with flag mistakes.
//
//nolint:paralleltest // Init registers flags on the package-level root command.
func TestExecuteRunErrorSkipsUsage(t *testing.T) {
	settings := filepath.Join(t.TempDir(), "speedtest-go.yaml")
	require.NoError(t, os.WriteFile(settings, []byte("debug: false\n"), 0o600))

	out, err := execute(t, "--config", settings, "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, `unknown ping mode "udp"`)
	assert.NotContains(t, out, "Usage:")
}

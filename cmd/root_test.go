package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
)

// execute runs a new root command with args and returns its error and combined output. See [executeContext].
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	return executeContext(t.Context(), t, args...)
}

// executeContext runs a new root command with ctx and args and returns its error and combined output. It resets the
// global viper state the command binds to, and always passes an unreachable --base-url, so no case can reach
// speedtest.net.
func executeContext(ctx context.Context, t *testing.T, args ...string) (string, error) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	var out bytes.Buffer

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append(args, "--base-url", "http://127.0.0.1:9"))

	err := rootCmd.ExecuteContext(ctx)

	return out.String(), err
}

// writeSettings writes an empty settings file and returns its path, so a run never reads the real home directory.
func writeSettings(t *testing.T) string {
	t.Helper()

	settings := filepath.Join(t.TempDir(), "speedtest-go.yaml")
	require.NoError(t, os.WriteFile(settings, []byte("debug: false\n"), 0o600))

	return settings
}

// TestNewRootCommand checks that the tree has its subcommands and that each call builds a separate tree.
//
//nolint:paralleltest // NewRootCommand binds flags to the global viper.
func TestNewRootCommand(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	first := NewRootCommand()
	second := NewRootCommand()

	names := make([]string, 0, len(first.Commands()))
	for _, sub := range first.Commands() {
		names = append(names, sub.Name())
	}

	assert.ElementsMatch(t, []string{"list", "cities", "version"}, names)
	assert.NotSame(t, first, second)
	assert.NotSame(t, first.Flags().Lookup("thread"), second.Flags().Lookup("thread"))
}

// TestNewRootCommandStartsClean checks that a flag set during one run is not seen by the next command, in its own
// flags or through viper, which reads the most recently built tree.
//
//nolint:paralleltest // NewRootCommand binds flags to the global viper.
func TestNewRootCommandStartsClean(t *testing.T) {
	_, err := execute(t, "--config", writeSettings(t), "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)

	next := NewRootCommand()

	pingMode := next.Flags().Lookup("ping-mode")
	assert.False(t, pingMode.Changed)
	assert.Equal(t, "http", pingMode.Value.String())
	assert.Equal(t, "http", viper.GetString("ping-mode"))
}

// TestExecuteVersion checks a full successful run of a subcommand, including the pre-run hook that reads settings.
//
//nolint:paralleltest // execute resets the global viper.
func TestExecuteVersion(t *testing.T) {
	out, err := execute(t, "--config", writeSettings(t), "version")
	require.NoError(t, err)
	assert.Equal(t, version.String()+"\n", out)
}

// TestExecuteFlagErrorIsInvalidSetting checks that a flag value cobra cannot parse is reported as an invalid
// setting, so the CLI exits with code 2 for it, and that the usage text follows. Parsing fails before the pre-run
// hook reads the settings file.
//
//nolint:paralleltest // execute resets the global viper.
func TestExecuteFlagErrorIsInvalidSetting(t *testing.T) {
	out, err := execute(t, "--thread", "abc")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.Contains(t, out, `invalid argument "abc"`)
	assert.Contains(t, out, "Usage:")
}

// TestExecuteSettingsFileErrorIsInvalidSetting checks that a settings file that cannot be read is an invalid
// setting, so the CLI exits with code 2, and that the error is not followed by the usage text.
//
//nolint:paralleltest // execute resets the global viper.
func TestExecuteSettingsFileErrorIsInvalidSetting(t *testing.T) {
	out, err := execute(t, "--config", filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, "failed to read config file")
	assert.NotContains(t, out, "Usage:")
}

// TestExecuteRunErrorSkipsUsage checks that an error after flag parsing, here an unknown ping mode, is reported
// without the usage text, which only helps with flag mistakes.
//
//nolint:paralleltest // execute resets the global viper.
func TestExecuteRunErrorSkipsUsage(t *testing.T) {
	out, err := execute(t, "--config", writeSettings(t), "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, `unknown ping mode "udp"`)
	assert.NotContains(t, out, "Usage:")
}

// TestExecuteInterrupted checks that both commands run with the context they are executed with, so an interrupt ends
// them with the context's error, and that they say "Interrupted" in place of cobra's error message.
//
//nolint:paralleltest // executeContext resets the global viper.
func TestExecuteInterrupted(t *testing.T) {
	for _, args := range [][]string{{"list"}, {"--json"}} {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		out, err := executeContext(ctx, t, append(args, "--config", writeSettings(t))...)
		require.ErrorIs(t, err, context.Canceled, args)
		assert.Equal(t, "Interrupted\n", out, args)
	}
}

// TestExecuteJSON runs the speed test command against the fake API, as a script would: stdout holds exactly one
// JSON document, and nothing else.
//
//nolint:paralleltest // The command binds flags to the global viper.
func TestExecuteJSON(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	api := testserver.NewAPI(t)

	var stdout, stderr bytes.Buffer

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"--config", writeSettings(t),
		"--base-url", api.URL(),
		"--json", "--no-download", "--no-upload",
	})

	require.NoError(t, rootCmd.ExecuteContext(t.Context()))

	decoder := json.NewDecoder(&stdout)

	var document map[string]any
	require.NoError(t, decoder.Decode(&document))
	assert.False(t, decoder.More(), "stdout must hold one JSON document")
	assert.Contains(t, document, "servers")
	assert.Empty(t, stderr.String())
}

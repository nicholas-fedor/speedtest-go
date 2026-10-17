package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
)

// TestMain isolates the commands from the user's own settings: HOME and XDG_CONFIG_HOME point at a temporary
// directory, and SPEEDTEST_GO_ variables from the shell are cleared.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "speedtest-go-cmd")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, variable := range os.Environ() {
		if name, _, _ := strings.Cut(variable, "="); strings.HasPrefix(name, config.EnvPrefix+"_") {
			_ = os.Unsetenv(name)
		}
	}

	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	xdg.Reload()

	code := m.Run()

	_ = os.RemoveAll(home)

	os.Exit(code)
}

// execute runs a new root command with args and returns its error and combined output. See [executeContext].
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	return executeContext(t.Context(), t, args...)
}

// executeContext runs a new root command with ctx and args and returns its error and combined output. It always
// passes an unreachable --base-url, so no case can reach speedtest.net.
func executeContext(ctx context.Context, t *testing.T, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append(args, "--base-url", "http://127.0.0.1:9"))

	err := rootCmd.ExecuteContext(ctx)

	return out.String(), err
}

// writeSettings writes a settings file with the given content and returns its path.
func writeSettings(t *testing.T, content string) string {
	t.Helper()

	settings := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(settings, []byte(content), 0o600))

	return settings
}

// TestNewRootCommand checks that the tree has its subcommands and that each call builds a separate tree.
func TestNewRootCommand(t *testing.T) {
	t.Parallel()

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

// TestNewRootCommandStartsClean checks that a flag set during one run is not seen by the next command.
func TestNewRootCommandStartsClean(t *testing.T) {
	t.Parallel()

	_, err := execute(t, "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)

	pingMode := NewRootCommand().Flags().Lookup("ping-mode")
	assert.False(t, pingMode.Changed)
	assert.Equal(t, "http", pingMode.Value.String())
}

// TestExecuteVersion checks a full successful run of a subcommand.
func TestExecuteVersion(t *testing.T) {
	t.Parallel()

	out, err := execute(t, "version")
	require.NoError(t, err)
	assert.Equal(t, version.String()+"\n", out)
}

// TestExecuteVersionIgnoresSettingsFile checks that commands without settings, such as version, do not read the
// settings file, so a broken file cannot stop them.
func TestExecuteVersionIgnoresSettingsFile(t *testing.T) {
	t.Parallel()

	_, err := execute(t, "--config", writeSettings(t, "jsn: true\n"), "version")
	require.NoError(t, err)
}

// TestExecuteFlagErrorIsInvalidSetting checks that a flag value cobra cannot parse is reported as an invalid
// setting, so the CLI exits with code 2 for it, and that the usage text follows.
func TestExecuteFlagErrorIsInvalidSetting(t *testing.T) {
	t.Parallel()

	out, err := execute(t, "--thread", "abc")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.Contains(t, out, `invalid argument "abc"`)
	assert.Contains(t, out, "Usage:")
}

// TestExecuteSettingsFileErrorIsInvalidSetting checks that a settings file that cannot be read or decoded is an
// invalid setting, so the CLI exits with code 2, and that the error names the file and is not followed by the usage
// text.
func TestExecuteSettingsFileErrorIsInvalidSetting(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"missing":     filepath.Join(t.TempDir(), "missing.yaml"),
		"unknown key": writeSettings(t, "jsn: true\n"),
	} {
		out, err := execute(t, "--config", path)
		require.ErrorIs(t, err, config.ErrInvalidSetting, name)
		require.ErrorIs(t, err, config.ErrInvalidFile, name)
		assert.Contains(t, out, path, name)
		assert.NotContains(t, out, "Usage:", name)
	}
}

// TestExecuteRunErrorSkipsUsage checks that an error after flag parsing, here an unknown ping mode, is reported
// without the usage text, which only helps with flag mistakes.
func TestExecuteRunErrorSkipsUsage(t *testing.T) {
	t.Parallel()

	out, err := execute(t, "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, `--ping-mode: unknown ping mode "udp"`)
	assert.NotContains(t, out, "Usage:")
}

// TestExecuteSettingsFile checks that a command reads its settings from the file --config names, and that a flag
// given on the command line wins over it.
func TestExecuteSettingsFile(t *testing.T) {
	t.Parallel()

	settings := writeSettings(t, "pingMode: udp\n")

	out, err := execute(t, "--config", settings)
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, settings+": pingMode", "the error names the file and key")

	_, err = execute(
		t,
		"--config",
		settings,
		"--ping-mode",
		"icmp",
		"--json",
		"--custom-url",
		"http://127.0.0.1:9",
	)
	require.NotErrorIs(
		t,
		err,
		config.ErrInvalidSetting,
		"the flag replaces the file's invalid value",
	)
}

// TestExecuteInterrupted checks that both commands run with the context they are executed with, so an interrupt ends
// them with the context's error, and that they say "Interrupted" in place of cobra's error message.
func TestExecuteInterrupted(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"list"}, {"--json"}} {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		out, err := executeContext(ctx, t, args...)
		require.ErrorIs(t, err, context.Canceled, args)
		assert.Equal(t, "Interrupted\n", out, args)
	}
}

// TestExecuteJSON runs the speed test command against the fake API, as a script would: stdout holds exactly one
// JSON document, and nothing else.
func TestExecuteJSON(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	var stdout, stderr bytes.Buffer

	rootCmd := NewRootCommand()
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"--base-url", api.URL(), "--json", "--no-download", "--no-upload"})

	require.NoError(t, rootCmd.ExecuteContext(t.Context()))

	decoder := json.NewDecoder(&stdout)

	var document map[string]any
	require.NoError(t, decoder.Decode(&document))
	assert.False(t, decoder.More(), "stdout must hold one JSON document")
	assert.Contains(t, document, "servers")
	assert.Empty(t, stderr.String())
}

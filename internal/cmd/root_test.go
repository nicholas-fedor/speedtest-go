// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	citiesmocks "github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/cities/mocks"
	listmocks "github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/list/mocks"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/mocks"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// errNetwork stands in for a failure of a speed test.
var errNetwork = errors.New("network down")

// testDependencies returns dependencies whose services are mocks that fail the test when called unexpectedly, with
// no environment variables and no default settings file, so nothing outside the test reaches the commands.
func testDependencies(t *testing.T) (Dependencies, *mocks.MockSpeedTester, *listmocks.MockLister) {
	t.Helper()

	tester := mocks.NewMockSpeedTester(t)
	lister := listmocks.NewMockLister(t)

	return Dependencies{
		SpeedTester: tester,
		Lister:      lister,
		Cities:      citiesmocks.NewMockWriter(t),
		Stdout:      io.Discard,
		LookupEnv:   env(nil),
		Version:     "speedtest-go v1.2.3",
		Dirs:        config.Dirs{ConfigHome: ""},
	}, tester, lister
}

// env returns a LookupEnv that reads only the given variables.
func env(variables map[string]string) config.LookupEnv {
	return func(key string) (string, bool) {
		value, ok := variables[key]

		return value, ok
	}
}

// execute runs a new root command built from deps with args and returns its error and combined output: the
// commands' output, cobra's help and usage text, and the error messages.
func execute(t *testing.T, deps Dependencies, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	deps.Stdout = &out
	root := NewRoot(deps)
	root.SetOut(&out)
	root.SetErr(&out)
	// A nil slice would make cobra read the test binary's own arguments.
	root.SetArgs(append([]string{}, args...))

	err := root.ExecuteContext(t.Context())

	return out.String(), err
}

// writeSettings writes a settings file with the given content and returns its path.
func writeSettings(t *testing.T, content string) string {
	t.Helper()

	settings := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(settings, []byte(content), 0o600))

	return settings
}

// TestNewRoot checks that the tree has its subcommands and that each call builds a separate tree.
func TestNewRoot(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)
	first := NewRoot(deps)
	second := NewRoot(deps)

	names := make([]string, 0, len(first.Commands()))
	for _, sub := range first.Commands() {
		names = append(names, sub.Name())
	}

	assert.ElementsMatch(t, []string{"list", "cities", "version"}, names)
	assert.NotSame(t, first, second)
	assert.NotSame(t, first.Flags().Lookup("thread"), second.Flags().Lookup("thread"))
}

// TestNewRootStartsClean checks that a flag set during one run is not seen by the next command.
func TestNewRootStartsClean(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)

	_, err := execute(t, deps, "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)

	pingMode := NewRoot(deps).Flags().Lookup("ping-mode")
	assert.False(t, pingMode.Changed)
	assert.Equal(t, "http", pingMode.Value.String())
}

// TestRootRunsSpeedTest checks that the root command passes the settings from its flags and the environment, and
// its output, to the speed tester.
func TestRootRunsSpeedTest(t *testing.T) {
	t.Parallel()

	deps, tester, _ := testDependencies(t)
	deps.LookupEnv = env(map[string]string{"SPEEDTEST_GO_UNIT": "binary-bits"})

	var out bytes.Buffer

	tester.EXPECT().
		Test(mock.Anything, mock.Anything, &out).
		RunAndReturn(func(_ context.Context, cfg config.Config, _ io.Writer) error {
			assert.Equal(t, speedtest.TCP, cfg.PingMode)
			assert.Equal(t, speedtest.UnitTypeBinaryBits, cfg.Unit)

			return nil
		}).
		Once()

	deps.Stdout = &out
	root := NewRoot(deps)
	root.SetArgs([]string{"--ping-mode", "tcp"})
	require.NoError(t, root.ExecuteContext(t.Context()))
}

// TestRootReadsDefaultSettingsFile checks that the root command reads the settings file in the user's config
// directory.
func TestRootReadsDefaultSettingsFile(t *testing.T) {
	t.Parallel()

	deps, tester, _ := testDependencies(t)
	deps.Dirs = config.Dirs{ConfigHome: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Dir(deps.Dirs.SettingsFile()), 0o700))
	require.NoError(t, os.WriteFile(deps.Dirs.SettingsFile(), []byte("pingMode: tcp\n"), 0o600))

	tester.EXPECT().
		Test(mock.Anything, mock.MatchedBy(func(cfg config.Config) bool {
			return cfg.PingMode == speedtest.TCP
		}), mock.Anything).
		Return(nil).
		Once()

	_, err := execute(t, deps)
	require.NoError(t, err)
}

// TestRootVersionIgnoresSettingsFile checks that commands without settings, such as version, do not read the
// settings file, so a broken file cannot stop them.
func TestRootVersionIgnoresSettingsFile(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)

	out, err := execute(t, deps, "--config", writeSettings(t, "jsn: true\n"), "version")
	require.NoError(t, err)
	assert.Equal(t, "speedtest-go v1.2.3\n", out)
}

// TestRootFlagErrorIsInvalidSetting checks that a flag value cobra cannot parse is reported as an invalid setting,
// so the CLI exits with code 2 for it, and that the usage text follows.
func TestRootFlagErrorIsInvalidSetting(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)

	out, err := execute(t, deps, "--thread", "abc")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.Contains(t, out, `invalid argument "abc"`)
	assert.Contains(t, out, "Usage:")
}

// TestRootSettingsFileErrorIsInvalidSetting checks that a settings file that cannot be read or decoded is an invalid
// setting, so the CLI exits with code 2, and that the error names the file and is not followed by the usage text.
func TestRootSettingsFileErrorIsInvalidSetting(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)

	for name, path := range map[string]string{
		"missing":     filepath.Join(t.TempDir(), "missing.yaml"),
		"unknown key": writeSettings(t, "jsn: true\n"),
	} {
		out, err := execute(t, deps, "--config", path)
		require.ErrorIs(t, err, config.ErrInvalidSetting, name)
		require.ErrorIs(t, err, config.ErrInvalidFile, name)
		assert.Contains(t, out, path, name)
		assert.NotContains(t, out, "Usage:", name)
	}
}

// TestRootRunErrorSkipsUsage checks that an error after flag parsing, here an unknown ping mode, is reported
// without the usage text, which only helps with flag mistakes.
func TestRootRunErrorSkipsUsage(t *testing.T) {
	t.Parallel()

	deps, _, _ := testDependencies(t)

	out, err := execute(t, deps, "--ping-mode", "udp")
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, `--ping-mode: unknown ping mode "udp"`)
	assert.NotContains(t, out, "Usage:")
}

// TestRootSettingsFile checks that a command reads its settings from the file --config names, and that a flag
// given on the command line wins over it.
func TestRootSettingsFile(t *testing.T) {
	t.Parallel()

	deps, tester, _ := testDependencies(t)
	settings := writeSettings(t, "pingMode: udp\n")

	out, err := execute(t, deps, "--config", settings)
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	assert.Contains(t, out, settings+": pingMode", "the error names the file and key")

	tester.EXPECT().
		Test(mock.Anything, mock.MatchedBy(func(cfg config.Config) bool {
			return cfg.PingMode == speedtest.TCP
		}), mock.Anything).
		Return(nil).
		Once()

	_, err = execute(t, deps, "--config", settings, "--ping-mode", "tcp")
	require.NoError(t, err, "the flag replaces the file's invalid value")
}

// TestRootReportsErrors checks that a failed command keeps cobra's error message without the usage text, and that
// every command an interrupt ends says "Interrupted" in its place.
func TestRootReportsErrors(t *testing.T) {
	t.Parallel()

	interrupted := fmt.Errorf("failed to fetch servers: %w", context.Canceled)

	tests := []struct {
		err  error
		name string
		want string
		args []string
	}{
		{name: "speed test failure", args: nil, err: errNetwork, want: "Error: network down\n"},
		{name: "speed test interrupted", args: nil, err: interrupted, want: "Interrupted\n"},
		{
			name: "list failure",
			args: []string{"list"},
			err:  errNetwork,
			want: "Error: network down\n",
		},
		{name: "list interrupted", args: []string{"list"}, err: interrupted, want: "Interrupted\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps, tester, lister := testDependencies(t)
			if len(tt.args) == 0 {
				tester.EXPECT().
					Test(mock.Anything, mock.Anything, mock.Anything).
					Return(tt.err).
					Once()
			} else {
				lister.EXPECT().
					List(mock.Anything, mock.Anything, mock.Anything).
					Return(tt.err).
					Once()
			}

			out, err := execute(t, deps, tt.args...)
			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, out)
		})
	}
}

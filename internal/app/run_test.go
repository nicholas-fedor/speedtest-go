// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
)

// runArgs runs the CLI with args and the given environment variables, with no default settings file, and returns
// the exit code, stdout, and stderr.
func runArgs(
	ctx context.Context,
	variables map[string]string,
	args ...string,
) (int, string, string) {
	var stdout, stderr bytes.Buffer

	code := run(ctx, process{
		stdout: &stdout,
		stderr: &stderr,
		lookupEnv: func(key string) (string, bool) {
			value, ok := variables[key]

			return value, ok
		},
		args: append([]string{}, args...),
		dirs: config.Dirs{ConfigHome: ""},
	})

	return code, stdout.String(), stderr.String()
}

// TestRunJSON runs the speed test against the fake API, as a script would: it exits with 0, stdout holds exactly
// one JSON document, and stderr holds nothing.
func TestRunJSON(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	code, stdout, stderr := runArgs(
		t.Context(),
		nil,
		"--base-url", api.URL(), "--json", "--no-download", "--no-upload",
	)
	require.Equal(t, exitOK, code, stderr)

	decoder := json.NewDecoder(bytes.NewBufferString(stdout))

	var document map[string]any
	require.NoError(t, decoder.Decode(&document))
	assert.False(t, decoder.More(), "stdout must hold one JSON document")
	assert.Contains(t, document, "servers")
	assert.Empty(t, stderr)
}

// TestRunListCommand runs the list command against the fake API: stdout holds the list, and the process's stderr
// holds the count of servers found.
func TestRunListCommand(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	code, stdout, stderr := runArgs(t.Context(), nil, "list", "--base-url", api.URL())
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "Tokyo (Japan)")
	assert.NotContains(t, stdout, "Public Servers")
	assert.Equal(t, "Found 2 Public Servers\n", stderr)
}

// TestRunCities checks that the cities command writes to the given stdout.
func TestRunCities(t *testing.T) {
	t.Parallel()

	code, stdout, _ := runArgs(t.Context(), nil, "cities")
	require.Equal(t, exitOK, code)
	assert.Contains(t, stdout, "Available city labels")
}

// TestRunVersion checks that the version command prints the binary's version line.
func TestRunVersion(t *testing.T) {
	t.Parallel()

	code, stdout, _ := runArgs(t.Context(), nil, "version")
	require.Equal(t, exitOK, code)
	assert.Equal(t, version.String()+"\n", stdout)
}

// TestRunInvalidSetting checks that invalid settings from a flag, an environment variable, or the settings file the
// environment names exit with code 2 and name their source.
func TestRunInvalidSetting(t *testing.T) {
	t.Parallel()

	settings := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(settings, []byte("pingMode: udp\n"), 0o600))

	tests := []struct {
		variables map[string]string
		name      string
		want      string
		args      []string
	}{
		{name: "flag", args: []string{"--ping-mode", "udp"}, want: "--ping-mode"},
		{
			name:      "environment",
			variables: map[string]string{"SPEEDTEST_GO_PING_MODE": "udp"},
			want:      "SPEEDTEST_GO_PING_MODE",
		},
		{
			name:      "settings file",
			variables: map[string]string{"SPEEDTEST_GO_CONFIG": settings},
			want:      settings + ": pingMode",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, _, stderr := runArgs(t.Context(), tt.variables, tt.args...)
			assert.Equal(t, exitInvalidSetting, code)
			assert.Contains(t, stderr, tt.want)
		})
	}
}

// TestRunInterrupted checks that a command an interrupt ends exits with code 130 and says "Interrupted".
func TestRunInterrupted(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)

	for _, args := range [][]string{{"list"}, {"--json"}} {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		code, _, stderr := runArgs(ctx, nil, append(args, "--base-url", api.URL())...)
		assert.Equal(t, exitInterrupted, code, args)
		assert.Equal(t, "Interrupted\n", stderr, args)
	}
}

// TestExitCode checks that interrupted runs exit with 130, invalid settings with 2, even when wrapped or joined, and
// other failures with 1.
func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		name string
		want int
	}{
		{name: "success", err: nil, want: exitOK},
		{name: "failure", err: errors.New("network down"), want: exitFailure},
		{
			name: "invalid setting",
			err:  fmt.Errorf("failed to load config: %w", config.ErrInvalidSetting),
			want: exitInvalidSetting,
		},
		{
			name: "interrupted",
			err:  fmt.Errorf("failed to fetch servers: %w", context.Canceled),
			want: exitInterrupted,
		},
		{
			name: "joined invalid setting",
			err: errors.Join(
				errors.New("other"),
				fmt.Errorf("%w --unit", config.ErrInvalidSetting),
			),
			want: exitInvalidSetting,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, exitCode(tt.err))
		})
	}
}

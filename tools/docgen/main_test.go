// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runMainEnv makes the test binary act as docgen itself, so the integration
// tests can run the real main with real arguments.
const runMainEnv = "DOCGEN_RUN_MAIN"

// TestMain runs docgen's main when runMainEnv is set and the tests otherwise.
//
// Parameters:
//   - m: the test runner.
func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) == "1" {
		main()
	}

	os.Exit(m.Run())
}

// Test_run covers argument handling and the exit codes.
func Test_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       func(dir string) []string
		wantStderr string
		wantCode   int
		wantPages  bool
	}{
		{
			name:      "writes the reference",
			args:      func(dir string) []string { return []string{"-out", dir} },
			wantCode:  exitOK,
			wantPages: true,
		},
		{
			name:       "help",
			args:       func(string) []string { return []string{"-h"} },
			wantCode:   exitOK,
			wantStderr: "-out",
		},
		{
			name:       "unknown flag",
			args:       func(string) []string { return []string{"-bogus"} },
			wantCode:   exitUsage,
			wantStderr: "flag provided but not defined",
		},
		{
			name:       "positional argument",
			args:       func(dir string) []string { return []string{"-out", dir, "extra"} },
			wantCode:   exitUsage,
			wantStderr: "unexpected arguments",
		},
		{
			name: "unusable output directory",
			args: func(dir string) []string {
				blocker := filepath.Join(dir, "blocker")
				_ = os.WriteFile(blocker, nil, filePerms)

				return []string{"-out", filepath.Join(blocker, "cli")}
			},
			wantCode:   exitFailure,
			wantStderr: "create output directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			var stderr bytes.Buffer

			code := run(tt.args(dir), &stderr)

			assert.Equal(t, tt.wantCode, code)
			assert.Contains(t, stderr.String(), tt.wantStderr)

			if tt.wantPages {
				require.FileExists(t, filepath.Join(dir, pageName))
			}
		})
	}
}

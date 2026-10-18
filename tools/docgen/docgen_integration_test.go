// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// committedReference is the checked-in CLI reference, relative to this
// package.
const committedReference = "../../docs/content/usage/cli-reference"

// runDocgen runs docgen through the test binary, which TestMain turns into
// docgen when DOCGEN_RUN_MAIN is set.
//
// Parameters:
//   - t: the test.
//   - args: docgen's arguments.
//
// Returns:
//   - []byte: the combined output.
//   - error: the process error, an [*exec.ExitError] for a non-zero exit.
func runDocgen(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()

	command := exec.CommandContext(t.Context(), os.Args[0], args...)

	command.Env = append(os.Environ(), "DOCGEN_RUN_MAIN=1")

	return command.CombinedOutput()
}

// readTree reads every file under dir, keyed by slash-separated relative path.
//
// Parameters:
//   - t: the test.
//   - dir: the directory.
//
// Returns:
//   - map[string]string: the file contents.
func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()

	files := map[string]string{}

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		files[filepath.ToSlash(rel)] = string(content)

		return nil
	})
	require.NoError(t, err)

	return files
}

// TestDocgenIsDeterministic runs docgen twice and expects identical bytes, so
// regenerating the reference never produces a spurious diff.
func TestDocgenIsDeterministic(t *testing.T) {
	t.Parallel()

	first, second := t.TempDir(), t.TempDir()

	output, err := runDocgen(t, "-out", first)
	require.NoError(t, err, string(output))

	output, err = runDocgen(t, "-out", second)
	require.NoError(t, err, string(output))

	firstTree := readTree(t, first)

	assert.NotEmpty(t, firstTree)
	assert.Equal(t, firstTree, readTree(t, second))
}

// TestDocgenMatchesCommittedReference fails when the command line changed
// without regenerating the site's CLI reference with task docs.
func TestDocgenMatchesCommittedReference(t *testing.T) {
	t.Parallel()

	out := t.TempDir()

	output, err := runDocgen(t, "-out", out)
	require.NoError(t, err, string(output))

	assert.Equal(
		t,
		readTree(t, committedReference),
		readTree(t, out),
		"run task docs to regenerate the CLI reference",
	)
}

// TestDocgenRejectsBadArguments checks a usage error exits with status 2.
func TestDocgenRejectsBadArguments(t *testing.T) {
	t.Parallel()

	output, err := runDocgen(t, "-out", t.TempDir(), "stray")

	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "want an exit error, got %v", err)
	assert.Equal(t, 2, exitErr.ExitCode())
	assert.Contains(t, string(output), "unexpected arguments")
}

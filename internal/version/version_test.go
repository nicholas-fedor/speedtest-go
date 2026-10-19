// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package version

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests below set the package-level Version, CommitSHA, and BuildTime, so they stay serial.

// TestGet_ldflags checks that the link-time values win, with the leading v trimmed from the version.
func TestGet_ldflags(t *testing.T) { //nolint:paralleltest // mutates package-level ldflag vars
	t.Cleanup(resetVars)

	Version = "v1.2.3"
	CommitSHA = "abcdef0123456789"
	BuildTime = "2026-05-02T12:00:00Z"

	got := Get()

	assert.Equal(t, "1.2.3", got.Version)
	assert.Equal(t, "abcdef0123456789", got.CommitSHA)
	assert.Equal(t, "2026-05-02T12:00:00Z", got.BuildTime)
	assert.Equal(t, runtime.Version(), got.GoVersion)
}

// TestGet_defaults checks that every field has a value without link-time values, from the build info or a sentinel.
func TestGet_defaults(t *testing.T) { //nolint:paralleltest // mutates package-level ldflag vars
	t.Cleanup(resetVars)

	Version = ""
	CommitSHA = ""
	BuildTime = ""

	got := Get()

	assert.NotEmpty(t, got.Version)
	assert.NotEmpty(t, got.CommitSHA)
	assert.NotEmpty(t, got.BuildTime)
	assert.Equal(t, runtime.Version(), got.GoVersion)

	if got.Version != "dev" {
		assert.NotContains(t, got.Version, "(devel)")
		assert.False(t, strings.HasPrefix(got.Version, "v"))
	}
}

// TestString_release checks the version line of a release build, without the Go version's GOEXPERIMENT suffix.
func TestString_release(t *testing.T) { //nolint:paralleltest // mutates package-level ldflag vars
	t.Cleanup(resetVars)

	Version = "1.8.0"
	CommitSHA = "1559e47abcdef"
	BuildTime = "2026-05-02T15:04:05Z"

	got := String()
	wantSuffix := " " + cleanGoVersion(
		runtime.Version(),
	) + " " + runtime.GOOS + "/" + runtime.GOARCH

	require.True(t, strings.HasPrefix(got, "speedtest-go v1.8.0 "))
	assert.Contains(t, got, "(1559e47, 2026-05-02)")
	assert.True(t, strings.HasSuffix(got, wantSuffix), "got %q want suffix %q", got, wantSuffix)
	assert.NotContains(t, got, "-X:")
}

// TestString_gitDescribeOmitsDuplicateCommit checks that a git describe version keeps its commit out of the metadata.
//
//nolint:paralleltest // mutates package-level ldflag vars
func TestString_gitDescribeOmitsDuplicateCommit(t *testing.T) {
	t.Cleanup(resetVars)

	Version = "1.8.0-209-g8000595-dirty"
	CommitSHA = "8000595"
	BuildTime = "2026-07-25T15:04:05Z"

	got := String()

	assert.Contains(t, got, "speedtest-go v1.8.0-209-g8000595-dirty")
	assert.Contains(t, got, "(2026-07-25)")
	assert.NotContains(t, got, "8000595,")
	assert.NotContains(t, got, "(8000595)")
}

// TestString_omitsUnknownMeta checks that the version line has no metadata segment when the commit and build time are
// unknown.
//
//nolint:paralleltest // mutates package-level ldflag vars
func TestString_omitsUnknownMeta(t *testing.T) {
	t.Cleanup(resetVars)

	Version = "1.8.0"
	CommitSHA = "unknown"
	BuildTime = "unknown"

	got := String()

	assert.Equal(
		t,
		"speedtest-go v1.8.0 "+cleanGoVersion(
			runtime.Version(),
		)+" "+runtime.GOOS+"/"+runtime.GOARCH,
		got,
	)
}

// TestBanner checks the startup banner of a release build.
func TestBanner(t *testing.T) { //nolint:paralleltest // mutates package-level ldflag vars
	t.Cleanup(resetVars)

	Version = "1.8.0"
	CommitSHA = "1559e47abcdef"
	BuildTime = "2026-05-02T15:04:05Z"

	assert.Equal(t, "speedtest-go v1.8.0 (1559e47, 2026-05-02)", Banner())
}

// TestBanner_gitDescribe checks that the banner of a git describe version shows only the build date.
//
//nolint:paralleltest // mutates package-level ldflag vars
func TestBanner_gitDescribe(t *testing.T) {
	t.Cleanup(resetVars)

	Version = "1.8.0-209-g8000595-dirty"
	CommitSHA = "8000595abcdef"
	BuildTime = "2026-07-25T12:00:00Z"

	assert.Equal(t, "speedtest-go v1.8.0-209-g8000595-dirty (2026-07-25)", Banner())
}

// TestHelpers checks the helpers that normalize and shorten the metadata.
func TestHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1.2.3", normalizeVersion("v1.2.3"))
	assert.Equal(t, "1.2.3", normalizeVersion("1.2.3"))
	assert.Empty(t, usableModuleVersion("(devel)"))
	assert.Empty(t, usableModuleVersion(""))
	assert.Equal(t, "1.0.0", usableModuleVersion("v1.0.0"))
	assert.Equal(t, "abc1234", shortCommit("abc1234def"))
	assert.Equal(t, "short", shortCommit("short"))
	assert.Equal(t, "2026-05-02", shortDate("2026-05-02T12:00:00Z"))
	assert.Equal(t, "unknown", shortDate("unknown"))
	assert.Equal(t, "go1.26.5", cleanGoVersion("go1.26.5-X:nodwarf5"))
	assert.Equal(t, "go1.26.5", cleanGoVersion("go1.26.5"))
	assert.True(t, versionEmbedsCommit("1.8.0-209-g8000595-dirty", "8000595"))
	assert.False(t, versionEmbedsCommit("1.8.0", "8000595"))
}

// resetVars clears the link-time values after a test that sets them.
func resetVars() {
	Version = ""
	CommitSHA = ""
	BuildTime = ""
}

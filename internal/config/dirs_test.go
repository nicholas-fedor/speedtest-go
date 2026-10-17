// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDirs_SettingsFile checks the default settings file inside the config directory, and none without one.
func TestDirs_SettingsFile(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		filepath.Join("/home/user/.config", "speedtest-go", "config.yaml"),
		Dirs{ConfigHome: "/home/user/.config"}.SettingsFile(),
	)
	assert.Empty(t, Dirs{ConfigHome: ""}.SettingsFile())
}

// TestSystemDirs checks that the user's config directory comes from the XDG lookup.
func TestSystemDirs(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, SystemDirs().ConfigHome)
}

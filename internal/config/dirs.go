// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

// Dirs are the user directories the settings file is found in.
type Dirs struct {
	// ConfigHome is XDG_CONFIG_HOME, normally ~/.config on Linux.
	ConfigHome string
}

// Settings file location.
const (
	// appDirName is the program's directory inside ConfigHome.
	appDirName = "speedtest-go"
	// settingsFileName is the settings file inside the program's directory.
	settingsFileName = "config.yaml"
)

// SystemDirs returns the current user's directories.
//
// The adrg/xdg package resolves them from the XDG environment variables, with the specification's defaults and the
// platform's equivalents elsewhere.
//
// Returns:
//   - Dirs: the user's config directory.
func SystemDirs() Dirs {
	return Dirs{ConfigHome: xdg.ConfigHome}
}

// SettingsFile returns the default settings file.
//
// Returns:
//   - string: speedtest-go/config.yaml inside ConfigHome, or empty when ConfigHome is unknown.
func (d Dirs) SettingsFile() string {
	if d.ConfigHome == "" {
		return ""
	}

	return filepath.Join(d.ConfigHome, appDirName, settingsFileName)
}

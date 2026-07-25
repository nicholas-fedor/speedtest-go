// Package version provides CLI application version metadata.
package version

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// ldflags targets set by Makefile and GoReleaser.
var (
	version string
	commit  string
	date    string
)

// Info holds resolved application version metadata.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// Get returns resolved application version metadata.
//
// Resolution order per field:
//   - Version: ldflag, then debug.BuildInfo Main.Version, then "dev"
//   - Commit: ldflag, then vcs.revision, then "unknown"
//   - BuildDate: ldflag, then vcs.time, then "unknown"
//   - GoVersion: runtime.Version()
//
// Returns:
//   - Info: populated version metadata
func Get() Info {
	info := Info{
		Version:   normalizeVersion(version),
		Commit:    commit,
		BuildDate: date,
		GoVersion: runtime.Version(),
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return finalize(info)
	}

	if info.Version == "" {
		if resolved := usableModuleVersion(buildInfo.Main.Version); resolved != "" {
			info.Version = resolved
		}
	}

	settings := buildSettings(buildInfo)
	if info.Commit == "" {
		info.Commit = settings["vcs.revision"]
	}

	if info.BuildDate == "" {
		info.BuildDate = settings["vcs.time"]
	}

	return finalize(info)
}

// String returns the user-facing version line for the version command.
//
// Example:
//
//	speedtest-go v1.8.0 (8000595, 2026-05-02) go1.26.5 linux/amd64
//
// Returns:
//   - string: formatted version information
func String() string {
	info := Get()
	parts := []string{"speedtest-go v" + info.Version}

	if meta := formatMeta(info); meta != "" {
		parts = append(parts, meta)
	}

	parts = append(parts, cleanGoVersion(info.GoVersion), runtime.GOOS+"/"+runtime.GOARCH)

	return strings.Join(parts, " ")
}

// Banner returns the startup banner version line.
//
// Example:
//
//	speedtest-go v1.8.0 (8000595, 2026-05-02)
//
// Returns:
//   - string: formatted banner version information
func Banner() string {
	info := Get()
	base := "speedtest-go v" + info.Version

	if meta := formatMeta(info); meta != "" {
		return base + " " + meta
	}

	return base
}

// formatMeta builds the parenthetical commit/date segment.
//
// Omits unknown fields. Skips commit when the version string already embeds it
// (git describe forms such as 1.8.0-209-g8000595-dirty).
func formatMeta(info Info) string {
	fields := make([]string, 0, 2)

	short := shortCommit(info.Commit)
	if short != "unknown" && !versionEmbedsCommit(info.Version, short) {
		fields = append(fields, short)
	}

	built := shortDate(info.BuildDate)
	if built != "unknown" {
		fields = append(fields, built)
	}

	if len(fields) == 0 {
		return ""
	}

	return "(" + strings.Join(fields, ", ") + ")"
}

// versionEmbedsCommit reports whether version already contains the short commit
// (typical of git describe output: 1.8.0-12-gabcdef0).
func versionEmbedsCommit(ver, short string) bool {
	if short == "" || short == "unknown" {
		return false
	}

	return strings.Contains(ver, short)
}

// cleanGoVersion strips GOEXPERIMENT suffixes (e.g. go1.26.5-X:nodwarf5 → go1.26.5).
func cleanGoVersion(raw string) string {
	if before, _, ok := strings.Cut(raw, "-X:"); ok {
		return before
	}

	return raw
}

// finalize applies default sentinels for empty fields.
func finalize(info Info) Info {
	if info.Version == "" {
		info.Version = "dev"
	}

	if info.Commit == "" {
		info.Commit = "unknown"
	}

	if info.BuildDate == "" {
		info.BuildDate = "unknown"
	}

	return info
}

// normalizeVersion trims a single leading v from a version token.
func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "v")
}

// usableModuleVersion returns a normalized module version, or empty if unusable.
func usableModuleVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "(devel)" {
		return ""
	}

	return normalizeVersion(raw)
}

// buildSettings maps BuildInfo settings by key.
func buildSettings(buildInfo *debug.BuildInfo) map[string]string {
	settings := make(map[string]string, len(buildInfo.Settings))
	for _, setting := range buildInfo.Settings {
		settings[setting.Key] = setting.Value
	}

	return settings
}

// shortCommit shortens a commit hash to 7 characters when longer.
func shortCommit(raw string) string {
	if len(raw) > 7 {
		return raw[:7]
	}

	return raw
}

// shortDate keeps YYYY-MM-DD when a longer timestamp is present.
func shortDate(raw string) string {
	if len(raw) > 10 {
		return raw[:10]
	}

	return raw
}

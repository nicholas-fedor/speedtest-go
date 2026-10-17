// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// LookupEnv reads an environment variable, like [os.LookupEnv].
type LookupEnv = func(key string) (string, bool)

// sources are the places a setting's value comes from, in the order they are consulted.
type sources struct {
	// flags are the parsed flags of the command.
	flags *pflag.FlagSet
	// lookupEnv reads an environment variable.
	lookupEnv LookupEnv
	// file holds the settings file's values, keyed by flag name.
	file map[string]fileValue
	// path is the settings file's path, for messages.
	path string
}

// Resolve reads the settings of a command from, in order, the flags given on the command line, the SPEEDTEST_GO_
// environment variables, the settings file, and the defaults, and validates them.
//
// The settings file is the one --config or SPEEDTEST_GO_CONFIG names, which must exist, or else the default file
// in the user's config directory, which may be missing. Settings the command has no flag for, such as --json for
// list, still come from the environment and the settings file.
//
// Parameters:
//   - flagSet: the command's parsed flags, or nil for none.
//   - lookupEnv: reads an environment variable.
//   - dirs: the user's directories, which hold the default settings file.
//
// Returns:
//   - Config: the validated settings.
//   - error: every invalid setting joined, each wrapping [ErrInvalidSetting].
func Resolve(flagSet *pflag.FlagSet, lookupEnv LookupEnv, dirs Dirs) (Config, error) {
	all := sources{flags: flagSet, lookupEnv: lookupEnv, file: nil, path: ""}

	path, named := all.settingsFile(dirs)
	if path != "" {
		load := LoadOptionalFile
		if named {
			load = LoadFile
		}

		file, err := load(path)
		if err != nil {
			return Config{}, fmt.Errorf("%w: %w", ErrInvalidSetting, err)
		}

		all.file, all.path = file.values(), path
	}

	return all.resolve()
}

// EnvName returns the environment variable of a setting: the prefix, an underscore, and the flag name in upper case
// with hyphens as underscores, such as SPEEDTEST_GO_PING_MODE.
//
// Parameters:
//   - name: the flag name.
//
// Returns:
//   - string: the variable's name.
func EnvName(name string) string {
	return EnvPrefix + "_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// settingsFile returns the settings file to read.
//
// Returns:
//   - string: the path, or empty when there is none.
//   - bool: true when --config or SPEEDTEST_GO_CONFIG named it, so it must exist.
func (s sources) settingsFile(dirs Dirs) (string, bool) {
	if path, _ := s.text(flags.Config); path != "" {
		return path, true
	}

	return dirs.SettingsFile(), false
}

// text returns a setting's value and where it came from: a flag given on the command line, the environment, the
// settings file, or the flag's default.
//
// Parameters:
//   - name: the flag name.
//
// Returns:
//   - string: the value, or empty when nothing sets it and it has no default.
//   - string: the source, such as --thread, SPEEDTEST_GO_THREAD, or the file path and key.
//
//nolint:nonamedreturns // Same-type returns need names.
func (s sources) text(name string) (value, source string) {
	flag := s.flag(name)
	if flag != nil && flag.Changed {
		return flag.Value.String(), "--" + name
	}

	if value, ok := s.env(name); ok {
		return value, EnvName(name)
	}

	if value, ok := s.file[name]; ok {
		return value.text, s.path + ": " + value.key
	}

	if flag != nil {
		return flag.DefValue, "--" + name
	}

	return "", ""
}

// list returns a list setting's entries and where they came from, in the order of [sources.text]. An environment
// variable, or a flag that is not a list, holds the entries separated by commas.
//
// Parameters:
//   - name: the flag name.
//
// Returns:
//   - []string: the entries, or nil when nothing sets them.
//   - string: the source.
func (s sources) list(name string) ([]string, string) {
	flag := s.flag(name)
	if flag != nil && flag.Changed {
		if slice, ok := flag.Value.(pflag.SliceValue); ok {
			return slice.GetSlice(), "--" + name
		}

		return splitList(flag.Value.String()), "--" + name
	}

	if value, ok := s.env(name); ok {
		return splitList(value), EnvName(name)
	}

	if value, ok := s.file[name]; ok {
		return value.list, s.path + ": " + value.key
	}

	return nil, ""
}

// flag returns the command's flag of a setting.
//
// Parameters:
//   - name: the flag name.
//
// Returns:
//   - *pflag.Flag: the flag, or nil when the command has none.
func (s sources) flag(name string) *pflag.Flag {
	if s.flags == nil {
		return nil
	}

	return s.flags.Lookup(name)
}

// env reads a setting's environment variable.
//
// Parameters:
//   - name: the flag name.
//
// Returns:
//   - string: the value.
//   - bool: true when the variable is set.
func (s sources) env(name string) (string, bool) {
	if s.lookupEnv == nil {
		return "", false
	}

	return s.lookupEnv(EnvName(name))
}

// resolve reads and validates every setting.
//
// Returns:
//   - Config: the settings.
//   - error: every invalid setting joined, each wrapping [ErrInvalidSetting].
func (s sources) resolve() (Config, error) {
	var errs []error

	invalid := func(source string, err error) {
		errs = append(errs, fmt.Errorf("%w %s: %w", ErrInvalidSetting, source, err))
	}

	text := func(name string) string {
		value, _ := s.text(name)

		return value
	}

	boolean := func(name string) bool {
		value, source := s.text(name)
		if value == "" {
			return false
		}

		parsed, err := strconv.ParseBool(value)
		if err != nil {
			invalid(source, fmt.Errorf("%q is %w", value, errNotBoolean))
		}

		return parsed
	}

	cfg := Config{
		ServerIDs:     nil,
		CustomURL:     text(flags.CustomURL),
		SavingMode:    boolean(flags.SavingMode),
		JSONOutput:    boolean(flags.JSON),
		JSONLOutput:   boolean(flags.JSONL),
		UnixOutput:    boolean(flags.Unix),
		Location:      text(flags.Location),
		City:          text(flags.City),
		Proxy:         text(flags.Proxy),
		Source:        text(flags.Source),
		DNSBindSource: boolean(flags.DNSBindSource),
		Multi:         boolean(flags.Multi),
		Thread:        0,
		Search:        text(flags.Search),
		UserAgent:     text(flags.UserAgent),
		NoDownload:    boolean(flags.NoDownload),
		NoUpload:      boolean(flags.NoUpload),
		PingMode:      speedtest.HTTP,
		TestMode:      speedtest.HTTPTest,
		Unit:          speedtest.UnitTypeDefaultMbps,
		UnitSet:       true,
		Debug:         boolean(flags.Debug),
		BaseURL:       text(flags.BaseURL),
		DSCP:          0,
		DownloadSize:  0,
		UploadSize:    0,
		MaskISP:       boolean(flags.MaskISP),
		CountryCodes:  nil,
	}

	var err error

	value, source := s.text(flags.PingMode)

	cfg.PingMode, err = parser.ParseProto(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.Protocol)

	cfg.TestMode, err = parser.ParseTestMode(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.Unit)

	cfg.Unit, err = parser.ParseUnit(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.Thread)

	cfg.Thread, err = parseThread(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.DSCP)

	cfg.DSCP, err = parseDSCP(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.DownloadSize)

	cfg.DownloadSize, err = parseTransferSize(value, speedtest.CheckDownloadSize)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.UploadSize)

	cfg.UploadSize, err = parseTransferSize(value, speedtest.CheckUploadSize)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.Source)

	err = checkSource(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.BaseURL)

	err = checkBaseURL(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.City)

	err = checkCity(value)
	if err != nil {
		invalid(source, err)
	}

	value, source = s.text(flags.Location)

	err = checkCoordinates(value)
	if err != nil {
		invalid(source, err)
	}

	entries, source := s.list(flags.Server)

	cfg.ServerIDs, err = parseServerIDs(entries)
	if err != nil {
		invalid(source, err)
	}

	entries, source = s.list(flags.FilterCC)

	cfg.CountryCodes, err = parseCountryCodes(entries)
	if err != nil {
		invalid(source, err)
	}

	errs = append(errs, checkCombinations(cfg)...)

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

// splitList returns the entries of a list held as text, separated by commas, without surrounding spaces or empty
// entries.
//
// Parameters:
//   - value: the text.
//
// Returns:
//   - []string: the entries, or nil when there are none.
func splitList(value string) []string {
	var entries []string

	for entry := range strings.SplitSeq(value, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

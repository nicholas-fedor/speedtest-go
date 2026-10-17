// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// speedtestFlags returns the speed test command's flags, parsed from args.
func speedtestFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	command := &cobra.Command{Use: "speedtest-go"}
	flags.RegisterRootFlags(command)
	require.NoError(t, command.ParseFlags(args))

	return command.Flags()
}

// listFlags returns flags like the list command's: only the flags every command takes.
func listFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	root := &cobra.Command{Use: "speedtest-go"}
	flags.RegisterRootFlags(root)

	list := &cobra.Command{Use: "list"}
	root.AddCommand(list)
	require.NoError(t, list.ParseFlags(args))

	return list.Flags()
}

// env returns a LookupEnv that reads only the given variables.
func env(variables map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, ok := variables[key]

		return value, ok
	}
}

// writeSettingsFile writes a settings file into a new config directory and returns the directories.
func writeSettingsFile(t *testing.T, content string) Dirs {
	t.Helper()

	dirs := Dirs{ConfigHome: t.TempDir()}
	require.NoError(t, os.MkdirAll(filepath.Dir(dirs.SettingsFile()), 0o700))
	require.NoError(t, os.WriteFile(dirs.SettingsFile(), []byte(content), 0o600))

	return dirs
}

// TestResolve_Flags checks that the flags given on the command line become the settings.
func TestResolve_Flags(t *testing.T) {
	t.Parallel()

	cfg, err := Resolve(speedtestFlags(t,
		"--saving-mode", "--json", "--city", "newyork", "--proxy", "http://proxy:8080",
		"--source", "192.0.2.10", "--dns-bind-source", "--thread", "4", "--search", "keyword",
		"--ua", "test-agent", "--no-download", "--ping-mode", "icmp", "--unit", "binary-bytes",
		"--debug", "--mask-isp", "--base-url", "http://127.0.0.1:8080/mirror",
		"--dl-size", "2000", "--ul-size", "300",
	), env(nil), Dirs{})
	require.NoError(t, err)

	assert.True(t, cfg.SavingMode)
	assert.True(t, cfg.JSONOutput)
	assert.False(t, cfg.JSONLOutput)
	assert.False(t, cfg.UnixOutput)
	assert.Equal(t, "newyork", cfg.City)
	assert.Equal(t, "http://proxy:8080", cfg.Proxy)
	assert.Equal(t, "192.0.2.10", cfg.Source)
	assert.True(t, cfg.DNSBindSource)
	assert.False(t, cfg.Multi)
	assert.Equal(t, 4, cfg.Thread)
	assert.Equal(t, "keyword", cfg.Search)
	assert.Equal(t, "test-agent", cfg.UserAgent)
	assert.True(t, cfg.NoDownload)
	assert.False(t, cfg.NoUpload)
	assert.Equal(t, speedtest.ICMP, cfg.PingMode)
	assert.Equal(t, speedtest.UnitTypeBinaryBytes, cfg.Unit)
	assert.True(t, cfg.UnitSet)
	assert.True(t, cfg.Debug)
	assert.True(t, cfg.MaskISP)
	assert.Equal(t, "http://127.0.0.1:8080/mirror", cfg.BaseURL)
	assert.Equal(t, 2000, cfg.DownloadSize)
	assert.Equal(t, 300, cfg.UploadSize)
}

// TestResolve_Defaults checks the settings when nothing sets them, with the command's flags and with none.
func TestResolve_Defaults(t *testing.T) {
	t.Parallel()

	for name, flagSet := range map[string]*pflag.FlagSet{
		"speed test flags": speedtestFlags(t),
		"no flags":         nil,
	} {
		cfg, err := Resolve(flagSet, env(nil), Dirs{})
		require.NoError(t, err, name)

		assert.Equal(t, Config{
			ServerIDs:     nil,
			CustomURL:     "",
			SavingMode:    false,
			JSONOutput:    false,
			JSONLOutput:   false,
			UnixOutput:    false,
			Location:      "",
			City:          "",
			Proxy:         "",
			Source:        "",
			DNSBindSource: false,
			Multi:         false,
			Thread:        0,
			Search:        "",
			UserAgent:     "",
			NoDownload:    false,
			NoUpload:      false,
			PingMode:      speedtest.HTTP,
			TestMode:      speedtest.HTTPTest,
			Unit:          speedtest.UnitTypeDefaultMbps,
			UnitSet:       true,
			Debug:         false,
			BaseURL:       "",
			DSCP:          0,
			DownloadSize:  0,
			UploadSize:    0,
			MaskISP:       false,
			CountryCodes:  nil,
		}, cfg, name)
	}
}

// TestResolve_Order checks that a flag given on the command line wins over the environment, the environment over
// the settings file, and the settings file over the default.
func TestResolve_Order(t *testing.T) {
	t.Parallel()

	dirs := writeSettingsFile(t, "pingMode: icmp\nthread: 3\nunit: binary-bits\n")
	variables := env(map[string]string{"SPEEDTEST_GO_PING_MODE": "tcp", "SPEEDTEST_GO_THREAD": "5"})

	cfg, err := Resolve(speedtestFlags(t, "--ping-mode", "http"), variables, dirs)
	require.NoError(t, err)

	assert.Equal(t, speedtest.HTTP, cfg.PingMode, "the flag wins")
	assert.Equal(t, 5, cfg.Thread, "the environment wins over the file")
	assert.Equal(t, speedtest.UnitTypeBinaryBits, cfg.Unit, "the file wins over the default")
	assert.Equal(t, speedtest.HTTPTest, cfg.TestMode, "the default applies when nothing sets it")
}

// TestResolve_SettingsWithoutFlag checks that a command without a setting's flag, as list has no --json, still
// takes the setting from the environment and the settings file.
func TestResolve_SettingsWithoutFlag(t *testing.T) {
	t.Parallel()

	dirs := writeSettingsFile(t, "unix: true\n")

	cfg, err := Resolve(
		listFlags(t, "--city", "tokyo"),
		env(map[string]string{"SPEEDTEST_GO_THREAD": "2"}),
		dirs,
	)
	require.NoError(t, err)

	assert.Equal(t, "tokyo", cfg.City)
	assert.Equal(t, 2, cfg.Thread)
	assert.True(t, cfg.UnixOutput)
}

// TestResolve_Environment checks the environment variables: hyphens in a name become underscores, a list holds its
// entries separated by commas, and variables without the prefix, such as DEBUG, are ignored.
func TestResolve_Environment(t *testing.T) {
	t.Parallel()

	cfg, err := Resolve(speedtestFlags(t), env(map[string]string{
		"SPEEDTEST_GO_DNS_BIND_SOURCE": "true",
		"SPEEDTEST_GO_SERVER":          "1001, 1002",
		"SPEEDTEST_GO_FILTER_CC":       "",
		"SPEEDTEST_GO_CUSTOM_URL":      "",
		"DEBUG":                        "1",
		"UNIT":                         "binary-bits",
	}), Dirs{})
	require.NoError(t, err)

	assert.True(t, cfg.DNSBindSource)
	assert.Equal(t, []int{1001, 1002}, cfg.ServerIDs)
	assert.False(t, cfg.Debug)
	assert.Equal(t, speedtest.UnitTypeDefaultMbps, cfg.Unit)

	cfg, err = Resolve(
		speedtestFlags(t),
		env(map[string]string{"SPEEDTEST_GO_FILTER_CC": "gb,us"}),
		Dirs{},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"GB", "US"}, cfg.CountryCodes)
}

// TestResolve_ListFlags checks the list flags given on the command line, repeated or separated by commas.
func TestResolve_ListFlags(t *testing.T) {
	t.Parallel()

	cfg, err := Resolve(speedtestFlags(t, "--server", "1", "--server", "2"), env(nil), Dirs{})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, cfg.ServerIDs)

	cfg, err = Resolve(listFlags(t, "--filter-cc", "gb,ie"), env(nil), Dirs{})
	require.NoError(t, err)
	assert.Equal(t, []string{"GB", "IE"}, cfg.CountryCodes)
}

// TestResolve_TextListFlag checks that a list setting given on the command line through a flag that is not a list
// takes precedence over the environment, with its entries separated by commas.
func TestResolve_TextListFlag(t *testing.T) {
	t.Parallel()

	flagSet := pflag.NewFlagSet("speedtest-go", pflag.ContinueOnError)
	flagSet.String(flags.Server, "", "")
	require.NoError(t, flagSet.Parse([]string{"--server", "1, 2"}))

	cfg, err := Resolve(flagSet, env(map[string]string{EnvName(flags.Server): "3"}), Dirs{})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, cfg.ServerIDs)
}

// TestResolve_SettingsFile checks the settings file: the default one in the config directory, which may be missing,
// and one named by --config or SPEEDTEST_GO_CONFIG, which must exist.
func TestResolve_SettingsFile(t *testing.T) {
	t.Parallel()

	content := "json: true\nserver: [1001, 1002]\nfilterCC: []\nprotocol: tcp\n"

	t.Run("default file", func(t *testing.T) {
		t.Parallel()

		cfg, err := Resolve(speedtestFlags(t), env(nil), writeSettingsFile(t, content))
		require.NoError(t, err)

		assert.True(t, cfg.JSONOutput)
		assert.Equal(t, []int{1001, 1002}, cfg.ServerIDs)
		assert.Empty(t, cfg.CountryCodes)
		assert.Equal(t, speedtest.TCPTest, cfg.TestMode)
	})

	t.Run("coordinates with a space", func(t *testing.T) {
		t.Parallel()

		cfg, err := Resolve(
			speedtestFlags(t),
			env(nil),
			writeSettingsFile(t, "location: 40.7128, -74.0060\n"),
		)
		require.NoError(t, err)
		assert.Equal(t, "40.7128, -74.0060", cfg.Location)
	})

	t.Run("missing default file", func(t *testing.T) {
		t.Parallel()

		_, err := Resolve(speedtestFlags(t), env(nil), Dirs{ConfigHome: t.TempDir()})
		require.NoError(t, err)
	})

	named := writeSettingsFile(t, "unix: true\n").SettingsFile()

	t.Run("named by --config", func(t *testing.T) {
		t.Parallel()

		cfg, err := Resolve(
			speedtestFlags(t, "--config", named),
			env(nil),
			writeSettingsFile(t, content),
		)
		require.NoError(t, err)
		assert.True(t, cfg.UnixOutput)
		assert.False(t, cfg.JSONOutput, "the named file replaces the default one")
	})

	t.Run("named by the environment", func(t *testing.T) {
		t.Parallel()

		cfg, err := Resolve(
			speedtestFlags(t),
			env(map[string]string{"SPEEDTEST_GO_CONFIG": named}),
			Dirs{},
		)
		require.NoError(t, err)
		assert.True(t, cfg.UnixOutput)
	})

	t.Run("missing named file", func(t *testing.T) {
		t.Parallel()

		missing := filepath.Join(t.TempDir(), "missing.yaml")

		_, err := Resolve(speedtestFlags(t, "--config", missing), env(nil), Dirs{})
		require.ErrorIs(t, err, ErrInvalidSetting)
		require.ErrorIs(t, err, ErrInvalidFile)
		assert.Contains(t, err.Error(), "missing.yaml")
	})

	t.Run("invalid file", func(t *testing.T) {
		t.Parallel()

		_, err := Resolve(speedtestFlags(t), env(nil), writeSettingsFile(t, "json: [\n"))
		require.ErrorIs(t, err, ErrInvalidSetting)
		require.ErrorIs(t, err, ErrInvalidFile)
	})

	t.Run("invalid value names the file and key", func(t *testing.T) {
		t.Parallel()

		dirs := writeSettingsFile(t, "pingMode: udp\n")

		_, err := Resolve(speedtestFlags(t), env(nil), dirs)
		require.ErrorIs(t, err, parser.ErrUnknownPingMode)
		assert.Contains(t, err.Error(), dirs.SettingsFile()+": pingMode")
	})
}

// TestResolve_RejectsInvalidSettings checks that each invalid value fails with an error wrapping ErrInvalidSetting
// that names where the value came from. The values come from the environment, which, unlike typed flags, can hold
// anything.
func TestResolve_RejectsInvalidSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		settings map[string]string
		wantErr  error
		name     string
		want     string
	}{
		{
			name:     "ping mode",
			settings: map[string]string{flags.PingMode: "udp"},
			wantErr:  parser.ErrUnknownPingMode,
			want:     "SPEEDTEST_GO_PING_MODE",
		},
		{
			name:     "unit",
			settings: map[string]string{flags.Unit: "kilobits"},
			wantErr:  parser.ErrUnknownUnit,
			want:     "SPEEDTEST_GO_UNIT",
		},
		{
			name:     "negative thread",
			settings: map[string]string{flags.Thread: "-1"},
			wantErr:  errNegative,
			want:     "SPEEDTEST_GO_THREAD",
		},
		{
			name:     "non-numeric thread",
			settings: map[string]string{flags.Thread: "many"},
			wantErr:  errNotWholeNumber,
			want:     `SPEEDTEST_GO_THREAD: "many" is not a whole number`,
		},
		{
			name:     "not a boolean",
			settings: map[string]string{flags.JSON: "yes"},
			wantErr:  errNotBoolean,
			want:     `SPEEDTEST_GO_JSON: "yes" is not true or false`,
		},
		{
			name:     "unknown source",
			settings: map[string]string{flags.Source: "no-such-interface0"},
			wantErr:  errUnknownSource,
			want:     `SPEEDTEST_GO_SOURCE: "no-such-interface0" is neither an IP address nor a network interface`,
		},
		{
			name:     "DSCP out of range",
			settings: map[string]string{flags.DSCP: "64"},
			wantErr:  speedtest.ErrInvalidDSCP,
			want:     "SPEEDTEST_GO_DSCP: cannot mark connections: DSCP must be from 0 to 63: 64",
		},
		{
			name:     "non-numeric DSCP",
			settings: map[string]string{flags.DSCP: "ef"},
			wantErr:  errNotWholeNumber,
			want:     `SPEEDTEST_GO_DSCP: "ef" is not a whole number`,
		},
		{
			name:     "unknown protocol",
			settings: map[string]string{flags.Protocol: "quic"},
			wantErr:  parser.ErrUnknownProtocol,
			want:     "SPEEDTEST_GO_PROTOCOL",
		},
		{
			name: "proxy with TCP tests",
			settings: map[string]string{
				flags.Protocol: "tcp",
				flags.Proxy:    "http://proxy.example:8080",
			},
			wantErr: errProxyWithTCP,
			want:    "--proxy cannot be used with --protocol tcp",
		},
		{
			name:     "download size not in the table",
			settings: map[string]string{flags.DownloadSize: "800"},
			wantErr:  speedtest.ErrInvalidTransferSize,
			want:     "SPEEDTEST_GO_DL_SIZE: cannot use this size",
		},
		{
			name:     "non-numeric upload size",
			settings: map[string]string{flags.UploadSize: "big"},
			wantErr:  errNotWholeNumber,
			want:     `SPEEDTEST_GO_UL_SIZE: "big" is not a whole number`,
		},
		{
			name:     "non-numeric server ID",
			settings: map[string]string{flags.Server: "1001,one"},
			wantErr:  errNotWholeNumber,
			want:     `SPEEDTEST_GO_SERVER: "one" is not a whole number`,
		},
		{
			name:     "country code too long",
			settings: map[string]string{flags.FilterCC: "gbr"},
			wantErr:  errInvalidCountryCode,
			want:     `SPEEDTEST_GO_FILTER_CC: "GBR" is not a two-letter country code or auto`,
		},
		{
			name:     "country filter with servers",
			settings: map[string]string{flags.FilterCC: "gb", flags.Server: "1001"},
			wantErr:  errFilterWithServers,
			want:     "--filter-cc cannot be used with --server or --custom-url",
		},
		{
			name: "country filter with a custom URL",
			settings: map[string]string{
				flags.FilterCC:  "gb",
				flags.CustomURL: "http://speed.example",
			},
			wantErr: errFilterWithServers,
			want:    "--filter-cc cannot be used with --server or --custom-url",
		},
		{
			name:     "unknown city",
			settings: map[string]string{flags.City: "atlantis"},
			wantErr:  errUnknownCity,
			want:     `SPEEDTEST_GO_CITY: "atlantis" is not a city label, run the cities command to list them`,
		},
		{
			name:     "malformed location",
			settings: map[string]string{flags.Location: "60"},
			wantErr:  speedtest.ErrInvalidLocationInput,
			want:     "SPEEDTEST_GO_LOCATION",
		},
		{
			name:     "location out of range",
			settings: map[string]string{flags.Location: "91,0"},
			wantErr:  speedtest.ErrInvalidInput,
			want:     "SPEEDTEST_GO_LOCATION",
		},
		{
			name:     "city with location",
			settings: map[string]string{flags.City: "tokyo", flags.Location: "35,139"},
			wantErr:  errCityWithLocation,
			want:     "--city and --location both set the location",
		},
		{
			name:     "search with servers",
			settings: map[string]string{flags.Search: "fiber", flags.Server: "1001"},
			wantErr:  errLocationWithServers,
			want:     "cannot be used with --server or --custom-url",
		},
		{
			name: "city with a custom URL",
			settings: map[string]string{
				flags.City:      "tokyo",
				flags.CustomURL: "http://speed.example",
			},
			wantErr: errLocationWithServers,
			want:    "cannot be used with --server or --custom-url",
		},
		{
			name:     "json and jsonl",
			settings: map[string]string{flags.JSON: "true", flags.JSONL: "true"},
			wantErr:  ErrInvalidSetting,
			want:     "--json, --jsonl cannot be combined",
		},
		{
			name: "all output formats",
			settings: map[string]string{
				flags.JSON:  "true",
				flags.JSONL: "true",
				flags.Unix:  "true",
			},
			wantErr: ErrInvalidSetting,
			want:    "--json, --jsonl, --unix cannot be combined",
		},
		{
			name:     "base URL without a scheme",
			settings: map[string]string{flags.BaseURL: "mirror.example"},
			wantErr:  speedtest.ErrInvalidBaseURL,
			want:     "SPEEDTEST_GO_BASE_URL",
		},
		{
			name:     "base URL with a query",
			settings: map[string]string{flags.BaseURL: "https://mirror.example?key=value"},
			wantErr:  speedtest.ErrInvalidBaseURL,
			want:     "SPEEDTEST_GO_BASE_URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			variables := make(map[string]string, len(tt.settings))
			for name, value := range tt.settings {
				variables[EnvName(name)] = value
			}

			cfg, err := Resolve(speedtestFlags(t), env(variables), Dirs{})
			require.ErrorIs(t, err, ErrInvalidSetting)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.want)
			assert.Equal(t, Config{}, cfg)
		})
	}
}

// TestResolve_ReportsEveryInvalidSetting checks that every invalid setting is reported at once, so a user can fix
// them in one pass.
func TestResolve_ReportsEveryInvalidSetting(t *testing.T) {
	t.Parallel()

	_, err := Resolve(speedtestFlags(t, "--json", "--unix"), env(map[string]string{
		"SPEEDTEST_GO_PING_MODE": "udp",
		"SPEEDTEST_GO_UNIT":      "kilobits",
		"SPEEDTEST_GO_THREAD":    "-1",
		"SPEEDTEST_GO_BASE_URL":  "ftp://mirror.example",
	}), Dirs{})
	require.ErrorIs(t, err, parser.ErrUnknownPingMode)
	require.ErrorIs(t, err, parser.ErrUnknownUnit)
	require.ErrorIs(t, err, speedtest.ErrInvalidBaseURL)

	var joined interface{ Unwrap() []error }

	require.ErrorAs(t, err, &joined)
	assert.Len(t, joined.Unwrap(), 5)

	for _, each := range joined.Unwrap() {
		assert.ErrorIs(t, each, ErrInvalidSetting)
	}
}

// TestResolve_AcceptsSingleOutputFormat checks that each output format alone is valid.
func TestResolve_AcceptsSingleOutputFormat(t *testing.T) {
	t.Parallel()

	for _, name := range []string{flags.JSON, flags.JSONL, flags.Unix} {
		_, err := Resolve(speedtestFlags(t, "--"+name, "--thread", "0"), env(nil), Dirs{})
		require.NoError(t, err, name)
	}
}

// TestEnvName checks the environment variable of a setting.
func TestEnvName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "SPEEDTEST_GO_PING_MODE", EnvName(flags.PingMode))
	assert.Equal(t, "SPEEDTEST_GO_CONFIG", EnvName(flags.Config))
}

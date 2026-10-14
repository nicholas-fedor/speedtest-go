package config

import (
	"bytes"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// snapshotLogger restores the standard logger writer and flags after the test.
func snapshotLogger(t *testing.T) {
	t.Helper()

	writer := log.Writer()
	flags := log.Flags()

	t.Cleanup(func() {
		log.SetOutput(writer)
		log.SetFlags(flags)
	})
}

//nolint:paralleltest // Setup mutates global logger
func TestSetupSavingModeEnablesUnixOutput(t *testing.T) {
	snapshotLogger(t)

	cfg := Config{
		SavingMode:  true,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.True(
		t,
		result.UnixOutput,
		"Setup should enable UnixOutput when SavingMode is active and no structured output is selected",
	)
}

//nolint:paralleltest // Setup mutates global logger
func TestSetupDoesNotOverrideExplicitUnixOutput(t *testing.T) {
	snapshotLogger(t)

	cfg := Config{
		SavingMode: true,
		JSONOutput: true,
		UnixOutput: false,
	}

	result := Setup(cfg)

	assert.False(t, result.UnixOutput, "Setup should not override explicit JSONOutput")
}

func TestSetupDoesNotMutateWhenSavingModeInactive(t *testing.T) {
	t.Parallel()

	cfg := Config{
		SavingMode:  false,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.False(
		t,
		result.UnixOutput,
		"Setup should not modify UnixOutput when SavingMode is inactive",
	)
	assert.False(t, cfg.UnixOutput, "Setup should not mutate the caller's Config value")
}

//nolint:paralleltest // mutates global viper
func TestLoadReadsViperState(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("saving-mode", true)
	viper.Set("json", true)
	viper.Set("jsonl", false)
	viper.Set("unix", false)
	viper.Set("city", "newyork")
	viper.Set("proxy", "http://proxy:8080")
	viper.Set("source", "192.0.2.10")
	viper.Set("dns-bind-source", true)
	viper.Set("multi", false)
	viper.Set("thread", 4)
	viper.Set("search", "keyword")
	viper.Set("ua", "test-agent")
	viper.Set("no-download", true)
	viper.Set("no-upload", false)
	viper.Set("ping-mode", "icmp")
	viper.Set("unit", "binary-bytes")
	viper.Set("debug", true)
	viper.Set("mask-isp", true)
	viper.Set("base-url", "http://127.0.0.1:8080/mirror")

	cfg, err := Load()
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
}

//nolint:paralleltest // mutates global viper
func TestLoadReturnsDefaultsWhenViperEmpty(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Empty(t, cfg.ServerIDs)
	assert.Empty(t, cfg.CustomURL)
	assert.False(t, cfg.SavingMode)
	assert.False(t, cfg.JSONOutput)
	assert.False(t, cfg.JSONLOutput)
	assert.False(t, cfg.UnixOutput)
	assert.Empty(t, cfg.Location)
	assert.Empty(t, cfg.City)
	assert.Empty(t, cfg.Proxy)
	assert.Empty(t, cfg.Source)
	assert.False(t, cfg.DNSBindSource)
	assert.False(t, cfg.Multi)
	assert.Equal(t, 0, cfg.Thread)
	assert.Empty(t, cfg.Search)
	assert.Empty(t, cfg.UserAgent)
	assert.False(t, cfg.NoDownload)
	assert.False(t, cfg.NoUpload)
	assert.Equal(t, speedtest.HTTP, cfg.PingMode)
	assert.Equal(t, speedtest.UnitTypeDefaultMbps, cfg.Unit)
	assert.False(t, cfg.Debug)
	assert.Empty(t, cfg.BaseURL)
}

//nolint:paralleltest // mutates global viper
func TestInitViperWithConfigFile(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "speedtest-go*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString("json: true\n")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	viper.Reset()
	t.Cleanup(viper.Reset)

	err = InitViper(tmpFile.Name())
	require.NoError(t, err)

	assert.Equal(t, tmpFile.Name(), viper.ConfigFileUsed())
	assert.True(t, viper.GetBool("json"))
}

func TestInitViperWithDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper("")
	require.NoError(t, err)

	// When no config file exists, ConfigFileUsed returns empty string.
	assert.Empty(t, viper.ConfigFileUsed())
}

//nolint:paralleltest // mutates global viper
func TestInitViperPropagatesParseError(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("json: [\n"), 0o600))

	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper(cfgPath)
	require.ErrorIs(t, err, ErrInvalidSetting)
	assert.Contains(t, err.Error(), "failed to read config file")
}

//nolint:paralleltest // mutates global viper
func TestInitViperPropagatesMissingExplicitFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	err := InitViper(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.ErrorIs(t, err, ErrInvalidSetting)
	assert.Contains(t, err.Error(), "failed to read config file")
}

//nolint:paralleltest // mutates global logger
func TestSetupDoesNotSuppressLogInPlainMode(t *testing.T) {
	snapshotLogger(t)

	var buf bytes.Buffer

	log.SetOutput(&buf)

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("separator")

	assert.Contains(t, buf.String(), "separator",
		"Setup should not suppress the standard logger in plain human-readable mode",
	)
}

//nolint:paralleltest // mutates global logger
func TestSetupSuppressesLogInJSONMode(t *testing.T) {
	snapshotLogger(t)

	var buf bytes.Buffer

	log.SetOutput(&buf)

	cfg := Config{
		JSONOutput:  true,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when JSONOutput is enabled",
	)
}

//nolint:paralleltest // mutates global logger
func TestSetupSuppressesLogInJSONLMode(t *testing.T) {
	snapshotLogger(t)

	var buf bytes.Buffer

	log.SetOutput(&buf)

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: true,
		UnixOutput:  false,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when JSONLOutput is enabled",
	)
}

//nolint:paralleltest // mutates global logger
func TestSetupSuppressesLogInUnixMode(t *testing.T) {
	snapshotLogger(t)

	var buf bytes.Buffer

	log.SetOutput(&buf)

	cfg := Config{
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  true,
	}

	Setup(cfg)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when UnixOutput is enabled",
	)
}

//nolint:paralleltest // mutates global logger
func TestSetupSuppressesLogWhenSavingModeEnablesUnixOutput(t *testing.T) {
	snapshotLogger(t)

	var buf bytes.Buffer

	log.SetOutput(&buf)

	cfg := Config{
		SavingMode:  true,
		JSONOutput:  false,
		JSONLOutput: false,
		UnixOutput:  false,
	}

	result := Setup(cfg)

	assert.True(t, result.UnixOutput,
		"Setup should enable UnixOutput when SavingMode is active",
	)

	log.Println("should-not-appear")

	assert.Empty(t, buf.String(),
		"Setup should suppress the standard logger when SavingMode converts to UnixOutput",
	)
}

// TestLoadRejectsInvalidBaseURL checks that Load fails on a base URL the library cannot use, before any request is made.
//
//nolint:paralleltest // mutates global viper
func TestLoadRejectsInvalidBaseURL(t *testing.T) {
	for _, raw := range []string{"ftp://mirror.example", "mirror.example", "https://mirror.example?key=value"} {
		t.Run(raw, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			viper.Set("base-url", raw)

			_, err := Load()
			require.ErrorIs(t, err, speedtest.ErrInvalidBaseURL)
			require.ErrorIs(t, err, ErrInvalidSetting)
		})
	}
}

// TestLoadRejectsInvalidSettings checks that each invalid value fails Load with an error wrapping ErrInvalidSetting
// that names the setting.
//
//nolint:paralleltest // mutates global viper
func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		settings map[string]any
		wantErr  error
		name     string
		want     string
	}{
		{
			name:     "ping mode",
			settings: map[string]any{"ping-mode": "udp"},
			wantErr:  parser.ErrUnknownPingMode,
			want:     "--ping-mode",
		},
		{
			name:     "unit",
			settings: map[string]any{"unit": "kilobits"},
			wantErr:  parser.ErrUnknownUnit,
			want:     "--unit",
		},
		{
			name:     "negative thread",
			settings: map[string]any{"thread": -1},
			wantErr:  ErrInvalidSetting,
			want:     "--thread",
		},
		{
			name:     "non-numeric thread",
			settings: map[string]any{"thread": "four"},
			wantErr:  ErrInvalidSetting,
			want:     "--thread",
		},
		{
			name:     "unknown source",
			settings: map[string]any{"source": "no-such-interface0"},
			wantErr:  errUnknownSource,
			want:     `--source: "no-such-interface0" is neither an IP address nor a network interface`,
		},
		{
			name:     "DSCP out of range",
			settings: map[string]any{"dscp": 64},
			wantErr:  speedtest.ErrInvalidDSCP,
			want:     "--dscp: cannot mark connections: DSCP must be from 0 to 63: 64",
		},
		{
			name:     "non-numeric DSCP",
			settings: map[string]any{"dscp": "ef"},
			wantErr:  errNotWholeNumber,
			want:     `--dscp: "ef" is not a whole number`,
		},
		{
			name:     "unknown protocol",
			settings: map[string]any{"protocol": "quic"},
			wantErr:  parser.ErrUnknownProtocol,
			want:     "--protocol",
		},
		{
			name:     "proxy with TCP tests",
			settings: map[string]any{"protocol": "tcp", "proxy": "http://proxy.example:8080"},
			wantErr:  errProxyWithTCP,
			want:     "--proxy cannot be used with --protocol tcp",
		},
		{
			name:     "download size not in the table",
			settings: map[string]any{"dl-size": 800},
			wantErr:  speedtest.ErrInvalidTransferSize,
			want:     "--dl-size: cannot use this size",
		},
		{
			name:     "non-numeric upload size",
			settings: map[string]any{"ul-size": "big"},
			wantErr:  errNotWholeNumber,
			want:     `--ul-size: "big" is not a whole number`,
		},
		{
			name:     "country code too long",
			settings: map[string]any{"filter-cc": []string{"gbr"}},
			wantErr:  errInvalidCountryCode,
			want:     `--filter-cc: "GBR" is not a two-letter country code or auto`,
		},
		{
			name:     "country filter with servers",
			settings: map[string]any{"filter-cc": []string{"gb"}, "server": []int{1001}},
			wantErr:  errFilterWithServers,
			want:     "--filter-cc cannot be used with --server or --custom-url",
		},
		{
			name: "country filter with a custom URL",
			settings: map[string]any{
				"filter-cc":  []string{"gb"},
				"custom-url": "http://speed.example",
			},
			wantErr: errFilterWithServers,
			want:    "--filter-cc cannot be used with --server or --custom-url",
		},
		{
			name:     "unknown city",
			settings: map[string]any{"city": "atlantis"},
			wantErr:  errUnknownCity,
			want:     `--city: "atlantis" is not a city label, run the cities command to list them`,
		},
		{
			name:     "malformed location",
			settings: map[string]any{"location": "60"},
			wantErr:  speedtest.ErrInvalidLocationInput,
			want:     "--location",
		},
		{
			name:     "location out of range",
			settings: map[string]any{"location": "91,0"},
			wantErr:  speedtest.ErrInvalidInput,
			want:     "--location",
		},
		{
			name:     "location not a number",
			settings: map[string]any{"location": "NaN,0"},
			wantErr:  speedtest.ErrInvalidInput,
			want:     "--location",
		},
		{
			name:     "city with location",
			settings: map[string]any{"city": "tokyo", "location": "35,139"},
			wantErr:  errCityWithLocation,
			want:     "--city and --location both set the location",
		},
		{
			name:     "search with servers",
			settings: map[string]any{"search": "fiber", "server": []int{1001}},
			wantErr:  errLocationWithServers,
			want:     "cannot be used with --server or --custom-url",
		},
		{
			name:     "city with a custom URL",
			settings: map[string]any{"city": "tokyo", "custom-url": "http://speed.example"},
			wantErr:  errLocationWithServers,
			want:     "cannot be used with --server or --custom-url",
		},
		{
			name:     "json and jsonl",
			settings: map[string]any{"json": true, "jsonl": true},
			wantErr:  ErrInvalidSetting,
			want:     "--json, --jsonl cannot be combined",
		},
		{
			name:     "jsonl and unix",
			settings: map[string]any{"jsonl": true, "unix": true},
			wantErr:  ErrInvalidSetting,
			want:     "--jsonl, --unix cannot be combined",
		},
		{
			name:     "all output formats",
			settings: map[string]any{"json": true, "jsonl": true, "unix": true},
			wantErr:  ErrInvalidSetting,
			want:     "--json, --jsonl, --unix cannot be combined",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			for key, value := range tt.settings {
				viper.Set(key, value)
			}

			cfg, err := Load()
			require.ErrorIs(t, err, ErrInvalidSetting)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.want)
			assert.Equal(t, Config{}, cfg)
		})
	}
}

// TestLoadReportsEveryInvalidSetting checks that Load reports all invalid settings at once, so a user can fix them
// in one pass.
//
//nolint:paralleltest // mutates global viper
func TestLoadReportsEveryInvalidSetting(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("ping-mode", "udp")
	viper.Set("unit", "kilobits")
	viper.Set("thread", -1)
	viper.Set("base-url", "ftp://mirror.example")
	viper.Set("json", true)
	viper.Set("unix", true)

	_, err := Load()
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

// TestLoadAcceptsSingleOutputFormat checks that each output format alone, and zero connections, are valid.
//
//nolint:paralleltest // mutates global viper
func TestLoadAcceptsSingleOutputFormat(t *testing.T) {
	for _, key := range []string{"json", "jsonl", "unix"} {
		t.Run(key, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			viper.Set(key, true)
			viper.Set("thread", 0)

			_, err := Load()
			require.NoError(t, err)
		})
	}
}

// TestInitViperReadsPrefixedEnv checks that settings come from SPEEDTEST_GO_ variables, with hyphens in a setting
// name read as underscores, and that unprefixed variables such as DEBUG are ignored.
func TestInitViperReadsPrefixedEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SPEEDTEST_GO_PING_MODE", "tcp")
	t.Setenv("SPEEDTEST_GO_DNS_BIND_SOURCE", "true")
	t.Setenv("SPEEDTEST_GO_THREAD", "3")
	t.Setenv("DEBUG", "1")
	t.Setenv("UNIT", "binary-bits")

	viper.Reset()
	t.Cleanup(viper.Reset)

	require.NoError(t, InitViper(""))

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, speedtest.TCP, cfg.PingMode)
	assert.True(t, cfg.DNSBindSource)
	assert.Equal(t, 3, cfg.Thread)
	assert.False(t, cfg.Debug)
	assert.Equal(t, speedtest.UnitTypeDefaultMbps, cfg.Unit)
}

// TestInitViperRejectsInvalidEnv checks that a bad value from the environment is validated like a flag value.
func TestInitViperRejectsInvalidEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SPEEDTEST_GO_THREAD", "many")

	viper.Reset()
	t.Cleanup(viper.Reset)

	require.NoError(t, InitViper(""))

	_, err := Load()
	require.ErrorIs(t, err, ErrInvalidSetting)
	assert.Contains(t, err.Error(), `--thread: "many" is not a whole number`)
}

// TestSetupUnit checks that Setup keeps an explicit decimal-bits unit, which is the zero UnitType, and resolves the
// default Mbps unit only when no unit was resolved.
//
//nolint:paralleltest // Setup mutates the global logger
func TestSetupUnit(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want speedtest.UnitType
	}{
		{name: "unset", cfg: Config{}, want: speedtest.UnitTypeDefaultMbps},
		{
			name: "explicit decimal bits",
			cfg:  Config{Unit: speedtest.UnitTypeDecimalBits, UnitSet: true},
			want: speedtest.UnitTypeDecimalBits,
		},
		{
			name: "explicit default",
			cfg:  Config{Unit: speedtest.UnitTypeDefaultMbps, UnitSet: true},
			want: speedtest.UnitTypeDefaultMbps,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshotLogger(t)

			cfg := Setup(tt.cfg)

			assert.Equal(t, tt.want, cfg.Unit)
			assert.True(t, cfg.UnitSet)
		})
	}
}

// TestCheckSource checks that IP addresses and interface names with a usable address are accepted, with or without
// a network prefix, and that anything else is rejected.
func TestCheckSource(t *testing.T) {
	t.Parallel()

	interfaces, err := net.Interfaces()
	require.NoError(t, err)

	var name string

	for _, iface := range interfaces {
		_, err = speedtest.ResolveInterface(iface.Name)
		if err == nil {
			name = iface.Name

			break
		}
	}

	require.NotEmpty(t, name, "no interface has a usable address")

	for _, source := range []string{"", "192.0.2.10", "2001:db8::1", "tcp://192.0.2.10", name, "tcp://" + name} {
		require.NoError(t, checkSource(source), source)
	}

	for _, source := range []string{"no-such-interface0", "speedtest.example", "tcp://no-such-interface0"} {
		assert.ErrorIs(t, checkSource(source), errUnknownSource, source)
	}
}

// TestParseDSCP checks that an empty value means unmarked, and that valid code points are accepted where the
// platform can mark sockets and rejected where it cannot.
func TestParseDSCP(t *testing.T) {
	t.Parallel()

	dscp, err := parseDSCP("")
	require.NoError(t, err)
	assert.Zero(t, dscp)

	dscp, err = parseDSCP(" 0 ")
	require.NoError(t, err)
	assert.Zero(t, dscp)

	_, err = parseDSCP("-1")
	require.ErrorIs(t, err, speedtest.ErrInvalidDSCP)

	dscp, err = parseDSCP("46")
	if speedtest.CheckDSCP(46) != nil {
		require.ErrorIs(t, err, speedtest.ErrDSCPUnsupported)

		return
	}

	require.NoError(t, err)
	assert.Equal(t, 46, dscp)
}

// TestLoadTestMode checks that the protocol setting selects the test mode, with HTTP as the default.
//
//nolint:paralleltest // mutates global viper
func TestLoadTestMode(t *testing.T) {
	for _, tt := range []struct {
		protocol string
		want     speedtest.TestMode
	}{
		{protocol: "", want: speedtest.HTTPTest},
		{protocol: "http", want: speedtest.HTTPTest},
		{protocol: "tcp", want: speedtest.TCPTest},
	} {
		t.Run(tt.protocol, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			viper.Set("protocol", tt.protocol)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.TestMode)
		})
	}
}

// TestLoadTransferSizes checks that table sizes load as given and that empty settings mean the defaults.
//
//nolint:paralleltest // mutates global viper
func TestLoadTransferSizes(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Zero(t, cfg.DownloadSize)
	assert.Zero(t, cfg.UploadSize)

	viper.Set("dl-size", 2000)
	viper.Set("ul-size", "300")

	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, 2000, cfg.DownloadSize)
	assert.Equal(t, 300, cfg.UploadSize)
}

// TestParseCountryCodes checks normalization: upper case, comma-separated entries as an environment variable gives
// them, blanks and repeats dropped in order, auto kept, and anything else rejected.
func TestParseCountryCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []string
		want    []string
		wantErr bool
	}{
		{name: "none", entries: nil, want: nil},
		{name: "one", entries: []string{"gb"}, want: []string{"GB"}},
		{name: "comma separated", entries: []string{"gb, us"}, want: []string{"GB", "US"}},
		{
			name:    "repeats and blanks",
			entries: []string{"gb", "GB", ",", "jp"},
			want:    []string{"GB", "JP"},
		},
		{name: "auto", entries: []string{"Auto", "gb"}, want: []string{"AUTO", "GB"}},
		{name: "three letters", entries: []string{"gbr"}, wantErr: true},
		{name: "digit", entries: []string{"g1"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCountryCodes(tt.entries)
			if tt.wantErr {
				require.ErrorIs(t, err, errInvalidCountryCode)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestLoadCountryCodesFromEnv checks that a comma-separated environment variable gives every code.
func TestLoadCountryCodesFromEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SPEEDTEST_GO_FILTER_CC", "gb,us")

	viper.Reset()
	t.Cleanup(viper.Reset)

	require.NoError(t, InitViper(""))

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"GB", "US"}, cfg.CountryCodes)
}

// TestLoadReadsServerChoice checks the settings that choose servers directly, which cannot be combined with the
// location settings.
//
//nolint:paralleltest // mutates global viper
func TestLoadReadsServerChoice(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("server", []int{1, 2})
	viper.Set("custom-url", "http://example.com")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, []int{1, 2}, cfg.ServerIDs)
	assert.Equal(t, "http://example.com", cfg.CustomURL)
}

// TestLoadReadsCoordinates checks that a valid --location loads as given, including spaces after the comma.
//
//nolint:paralleltest // mutates global viper
func TestLoadReadsCoordinates(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("location", "40.7128, -74.0060")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "40.7128, -74.0060", cfg.Location)
}

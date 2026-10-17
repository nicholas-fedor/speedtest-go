package config

import (
	"bytes"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

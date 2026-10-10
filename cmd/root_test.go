package cmd

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// initOnce registers the commands and flags once per test binary, since repeated runs reuse the global root command.
var initOnce sync.Once

// TestExecuteFlagErrorIsInvalidSetting checks that a flag value cobra cannot parse is reported as an invalid
// setting, so the CLI exits with code 2 for it. Parsing fails before any initializer reads the settings file.
//
//nolint:paralleltest // Init registers flags on the package-level root command.
func TestExecuteFlagErrorIsInvalidSetting(t *testing.T) {
	initOnce.Do(Init)

	var out bytes.Buffer

	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--thread", "abc"})

	err := Execute()
	require.ErrorIs(t, err, config.ErrInvalidSetting)
	require.Contains(t, out.String(), `invalid argument "abc"`)
}

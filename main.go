// Package main is the entry point for the speedtest application.
package main

import (
	"errors"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/cmd"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// Exit codes returned by the CLI.
const (
	// exitOK reports success.
	exitOK = 0
	// exitFailure reports a failed run.
	exitFailure = 1
	// exitInvalidSetting reports a setting value the CLI rejected before running.
	exitInvalidSetting = 2
)

func main() {
	os.Exit(exitCode(cmd.NewRootCommand().Execute()))
}

// exitCode maps the result of a run to the process exit code.
//
// Parameters:
//   - err: the error from the command, or nil.
//
// Returns:
//   - int: 0 for success, 2 for an error wrapping [config.ErrInvalidSetting], and 1 otherwise.
func exitCode(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, config.ErrInvalidSetting):
		return exitInvalidSetting
	default:
		return exitFailure
	}
}

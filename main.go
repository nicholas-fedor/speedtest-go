// Package main is the entry point for the speedtest application.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

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
	// exitInterrupted reports a run that an interrupt ended, 128 plus the number of SIGINT, as shells report it.
	exitInterrupted = 130
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// The first interrupt cancels the run. Restoring the default handling lets a second one end the process at once
	// when the run does not stop.
	go func() {
		<-ctx.Done()
		stop()
	}()

	err := cmd.NewRootCommand().ExecuteContext(ctx)

	stop()

	os.Exit(exitCode(err))
}

// exitCode maps the result of a run to the process exit code.
//
// Parameters:
//   - err: the error from the command, or nil.
//
// Returns:
//   - int: 0 for success, 130 for an error wrapping [context.Canceled], which only an interrupt causes, 2 for an
//     error wrapping [config.ErrInvalidSetting], and 1 otherwise.
func exitCode(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, context.Canceled):
		return exitInterrupted
	case errors.Is(err, config.ErrInvalidSetting):
		return exitInvalidSetting
	default:
		return exitFailure
	}
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/cities"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/list"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
)

// services carries out what the commands ask for: the speed test, the server list, and the city list.
type services struct{}

// process holds what a run reads from the process, so tests can supply their own.
type process struct {
	// stdout receives the output of the commands. Cobra's help and usage text keep cobra's default streams.
	stdout io.Writer
	// stderr receives the error messages.
	stderr io.Writer
	// lookupEnv reads an environment variable.
	lookupEnv config.LookupEnv
	// dirs are the user's directories, which hold the default settings file.
	dirs config.Dirs
	// args are the command-line arguments, without the program name.
	args []string
}

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

// services must satisfy every command's contract.
var (
	_ cmd.SpeedTester = services{}
	_ list.Lister     = services{}
	_ cities.Writer   = services{}
)

// Run runs the command the process arguments name and returns the exit code.
//
// Parameters:
//   - ctx: ends the command. The CLI cancels it on an interrupt.
//
// Returns:
//   - int: the exit code, 0 for success, 1 for a failure, 2 for an invalid setting, or 130 for an interrupt.
func Run(ctx context.Context) int {
	return run(ctx, process{
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		lookupEnv: os.LookupEnv,
		args:      os.Args[1:],
		dirs:      config.SystemDirs(),
	})
}

// Test runs the speed test. See [runSpeedtest].
//
// Parameters:
//   - ctx: ends the run.
//   - cfg: the validated settings.
//   - out: receives the progress and the results.
//
// Returns:
//   - error: the error that ended the run.
func (services) Test(ctx context.Context, cfg config.Config, out io.Writer) error {
	return runSpeedtest(ctx, cfg, out)
}

// List writes the server list. See [runList].
//
// Parameters:
//   - ctx: ends the server lookups.
//   - cfg: the validated settings.
//   - out: receives the list.
//
// Returns:
//   - error: the error of a lookup.
func (services) List(ctx context.Context, cfg config.Config, out io.Writer) error {
	return runList(ctx, cfg, out)
}

// WriteCities writes the city list. See [writeCities].
//
// Parameters:
//   - out: receives the list.
//
// Returns:
//   - error: the error of writing the list.
func (services) WriteCities(out io.Writer) error {
	return writeCities(out)
}

// run builds the command tree, runs the command that process names, and returns the exit code.
//
// Parameters:
//   - ctx: ends the command.
//   - proc: the process's arguments, streams, environment, and directories.
//
// Returns:
//   - int: the exit code of the command's error. See [exitCode].
func run(ctx context.Context, proc process) int {
	//nolint:contextcheck // The commands read ctx from the command, which ExecuteContext sets below.
	root := cmd.NewRoot(cmd.Dependencies{
		SpeedTester: services{},
		Lister:      services{},
		Cities:      services{},
		Stdout:      proc.stdout,
		LookupEnv:   proc.lookupEnv,
		Version:     version.String(),
		Dirs:        proc.dirs,
	})
	root.SetArgs(proc.args)
	root.SetErr(proc.stderr)

	return exitCode(root.ExecuteContext(ctx))
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

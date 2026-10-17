// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"io"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/cities"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/list"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// SpeedTester runs the speed test.
type SpeedTester interface {
	// Test runs the speed test on the servers the settings select and writes its progress and results.
	//
	// Parameters:
	//   - ctx: ends the run. The CLI cancels it on an interrupt.
	//   - cfg: the validated settings.
	//   - out: receives the progress and the results.
	//
	// Returns:
	//   - error: the error that ended the run, which wraps the context's error when the context ended it.
	Test(ctx context.Context, cfg config.Config, out io.Writer) error
}

// Dependencies are what the command tree needs from the process: the services the commands call, where their output
// goes, the version line, and the sources of the settings besides the flags.
//
// Cobra's own help and usage text, and its error messages, go to the root command's output and error streams, set
// with [cobra.Command.SetOut] and [cobra.Command.SetErr].
type Dependencies struct {
	// SpeedTester runs the speed test for the root command.
	SpeedTester SpeedTester
	// Lister lists the servers for the list command.
	Lister list.Lister
	// Cities writes the city labels for the cities command.
	Cities cities.Writer
	// Stdout receives the output of the commands.
	Stdout io.Writer
	// LookupEnv reads the SPEEDTEST_GO_ environment variables.
	LookupEnv config.LookupEnv
	// Version is the line the version command prints.
	Version string
	// Dirs are the user's directories, which hold the default settings file.
	Dirs config.Dirs
}

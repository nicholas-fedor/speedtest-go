// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:generate go run github.com/nicholas-fedor/speedtest-go/v2/tools/docgen -out ../../docs/content/usage/cli-reference

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd"
)

// Exit codes.
const (
	// exitOK reports success.
	exitOK = 0
	// exitFailure reports a generation failure.
	exitFailure = 1
	// exitUsage reports invalid arguments.
	exitUsage = 2
)

// defaultOut is the output directory, relative to the repository root.
const defaultOut = "./docs/content/usage/cli-reference"

// main writes the CLI reference and exits with the result of run.
func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run parses arguments and writes the CLI reference.
//
// Parameters:
//   - args: command-line arguments without the program name.
//   - stderr: receives usage and error text.
//
// Returns:
//   - int: exitOK, exitFailure, or exitUsage.
func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("docgen", flag.ContinueOnError)
	flags.SetOutput(stderr)

	out := flags.String("out", defaultOut, "Output directory")

	err := flags.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}

		return exitUsage
	}

	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "docgen: unexpected arguments: %v\n", flags.Args())

		return exitUsage
	}

	generator, err := NewDocGenerator()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "docgen: %v\n", err)

		return exitFailure
	}

	err = generator.Generate(cmd.DocRoot(), *out)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "docgen: %v\n", err)

		return exitFailure
	}

	return exitOK
}

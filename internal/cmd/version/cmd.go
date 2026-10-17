// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package version

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// NewCommand builds the version command.
//
// Parameters:
//   - line: the version line to print.
//   - stdout: receives the line.
//
// Returns:
//   - *cobra.Command: the command that prints application version information.
func NewCommand(line string, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the application version, commit, build date, Go runtime, and platform.",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(stdout, line)
		},
	}
}

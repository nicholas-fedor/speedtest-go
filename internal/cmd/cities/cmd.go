// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package cities

import (
	"io"

	"github.com/spf13/cobra"
)

// Writer writes the city labels.
type Writer interface {
	// WriteCities writes the predefined city labels with their country codes and coordinates.
	//
	// Parameters:
	//   - out: receives the list.
	//
	// Returns:
	//   - error: the error of writing the list.
	WriteCities(out io.Writer) error
}

// NewCommand builds the cities command.
//
// Parameters:
//   - writer: writes the city labels.
//   - stdout: receives the list.
//
// Returns:
//   - *cobra.Command: the command that lists the predefined city labels.
func NewCommand(writer Writer, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "cities",
		Short: "List predefined city labels",
		Long:  "Display a list of predefined city labels that can be used for location-based server selection.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return writer.WriteCities(stdout)
		},
	}
}

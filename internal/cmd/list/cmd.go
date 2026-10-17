// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package list

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
)

// Settings reads a command's settings from its parsed flags and the other sources of settings.
type Settings = func(flagSet *pflag.FlagSet) (config.Config, error)

// Lister lists the servers.
type Lister interface {
	// List writes the servers the settings select.
	//
	// Parameters:
	//   - ctx: ends the server lookups. The CLI cancels it on an interrupt.
	//   - cfg: the validated settings.
	//   - out: receives the list.
	//
	// Returns:
	//   - error: the error of a lookup, which wraps the context's error when the context ended it.
	List(ctx context.Context, cfg config.Config, out io.Writer) error
}

// NewCommand builds the list command.
//
// Parameters:
//   - lister: lists the servers.
//   - settings: reads the command's settings.
//   - stdout: receives the list.
//
// Returns:
//   - *cobra.Command: the command that lists available speedtest servers.
func NewCommand(lister Lister, settings Settings, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available speedtest servers",
		Long:  "Display a list of available speedtest.net servers with their details.",
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := settings(command.Flags())
			if err != nil {
				return err
			}

			return lister.List(command.Context(), cfg, stdout)
		},
	}
}

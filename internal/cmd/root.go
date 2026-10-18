// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/cities"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/list"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd/version"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
)

// NewRoot builds the speedtest-go command tree with its subcommands and flags. Each call returns a separate tree, so
// a command can run once without state left over from another.
//
// Parameters:
//   - deps: the services the commands call and the sources of their settings.
//
// Returns:
//   - *cobra.Command: the root command, ready to execute.
func NewRoot(deps Dependencies) *cobra.Command {
	settings := resolver(deps)

	root := &cobra.Command{
		Use:   "speedtest-go",
		Short: "Test internet bandwidth using speedtest.net",
		Long:  "A command-line tool to test internet download and upload speeds using speedtest.net servers.",
		// Flags have parsed by the time this runs, so later errors are not usage mistakes and skip the usage text.
		PersistentPreRun: func(command *cobra.Command, _ []string) {
			command.SilenceUsage = true
		},
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := settings(command.Flags())
			if err != nil {
				return err
			}

			return deps.SpeedTester.Test(command.Context(), cfg, deps.Stdout)
		},
	}

	// Flag values cobra cannot parse, such as a non-numeric --thread, are invalid settings.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %w", config.ErrInvalidSetting, err)
	})

	flags.RegisterRootFlags(root)

	root.AddCommand(
		list.NewCommand(deps.Lister, settings, deps.Stdout),
		cities.NewCommand(deps.Cities, deps.Stdout),
		version.NewCommand(deps.Version, deps.Stdout),
	)

	reportInterrupts(root)

	return root
}

// DocRoot returns the command tree for documentation generation.
//
// tools/docgen reads command metadata only, so the tree carries no dependencies. Every run function is built but
// never invoked, and building the real tree keeps the documented commands and flags identical to the shipped ones.
//
// Returns:
//   - *cobra.Command: the root command, safe to inspect.
func DocRoot() *cobra.Command {
	return NewRoot(Dependencies{
		SpeedTester: nil,
		Lister:      nil,
		Cities:      nil,
		Stdout:      nil,
		LookupEnv:   nil,
		Version:     "",
		Dirs:        config.Dirs{ConfigHome: ""},
	})
}

// resolver returns the function that reads a command's settings from its flags, the environment, and the settings
// file.
//
// Parameters:
//   - deps: the sources of the settings besides the flags.
//
// Returns:
//   - list.Settings: the function.
func resolver(deps Dependencies) list.Settings {
	return func(flagSet *pflag.FlagSet) (config.Config, error) {
		cfg, err := config.Resolve(flagSet, deps.LookupEnv, deps.Dirs)
		if err != nil {
			return config.Config{}, fmt.Errorf("failed to load config: %w", err)
		}

		return cfg, nil
	}
}

// reportInterrupts makes every command in the tree print "Interrupted" in place of cobra's error message when an
// interrupt ended it, so every interrupted command ends the same way. Other errors keep cobra's message.
//
// Parameters:
//   - command: the root of the tree.
func reportInterrupts(command *cobra.Command) {
	if run := command.RunE; run != nil {
		command.RunE = func(command *cobra.Command, args []string) error {
			err := run(command, args)
			if errors.Is(err, context.Canceled) {
				command.SilenceErrors = true
				_, _ = fmt.Fprintln(command.ErrOrStderr(), "Interrupted")
			}

			return err
		}
	}

	for _, sub := range command.Commands() {
		reportInterrupts(sub)
	}
}

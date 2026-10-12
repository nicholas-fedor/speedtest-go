// Package cmd provides CLI commands for the speedtest application.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/app"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
)

// NewRootCommand builds the speedtest-go command tree with its subcommands and flags. Each call returns a
// separate tree, so a command can run once without state left over from another.
//
// Returns:
//   - *cobra.Command: the root command, ready to execute.
func NewRootCommand() *cobra.Command {
	cfgFile := ""

	rootCmd := &cobra.Command{
		Use:   "speedtest-go",
		Short: "Test internet bandwidth using speedtest.net",
		Long:  "A command-line tool to test internet download and upload speeds using speedtest.net servers.",
		// Flags have parsed by the time this runs, so later errors are not usage mistakes and skip the usage text.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			return config.InitViper(cfgFile)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			return app.RunSpeedtest(cfg)
		},
	}

	// Flag values cobra cannot parse, such as a non-numeric --thread, are invalid settings.
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %w", config.ErrInvalidSetting, err)
	})

	flags.RegisterRootFlags(rootCmd, &cfgFile)

	rootCmd.AddCommand(
		newListCommand(),
		newCitiesCommand(),
		newVersionCommand(),
	)

	return rootCmd
}

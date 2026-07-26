// Package cmd provides CLI commands for the speedtest application.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/internal/app"
	"github.com/nicholas-fedor/speedtest-go/internal/config"
	"github.com/nicholas-fedor/speedtest-go/internal/flags"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "speedtest-go",
	Short: "Test internet bandwidth using speedtest.net",
	Long:  "A command-line tool to test internet download and upload speeds using speedtest.net servers.",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return app.RunSpeedtest(cfg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute root command: %w", err)
	}

	return nil
}

// Init initializes the CLI commands and flags.
func Init() {
	cfgFile := ""

	cobra.OnInitialize(func() {
		cobra.CheckErr(config.InitViper(cfgFile))
	})

	flags.RegisterRootFlags(rootCmd, &cfgFile)

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(citiesCmd)
	rootCmd.AddCommand(versionCmd)

	flags.RegisterListFlags(listCmd)
}

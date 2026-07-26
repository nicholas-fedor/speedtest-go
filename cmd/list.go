package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/internal/app"
	"github.com/nicholas-fedor/speedtest-go/internal/config"
)

// listCmd represents the list command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available speedtest servers",
	Long:  "Display a list of available speedtest.net servers with their details.",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return app.RunList(cfg)
	},
}

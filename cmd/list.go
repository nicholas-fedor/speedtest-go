package cmd

import (
	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/app"
)

// newListCommand builds the list command with its flags.
//
// Returns:
//   - *cobra.Command: the command that lists available speedtest servers.
func newListCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available speedtest servers",
		Long:  "Display a list of available speedtest.net servers with their details.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolveSettings(cmd)
			if err != nil {
				return err
			}

			return reportInterrupt(cmd, app.RunList(cmd.Context(), cfg, cmd.OutOrStdout()))
		},
	}

	return listCmd
}

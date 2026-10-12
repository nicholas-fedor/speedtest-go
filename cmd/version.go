package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
)

// newVersionCommand builds the version command.
//
// Returns:
//   - *cobra.Command: the command that prints application version information.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the application version, commit, build date, Go runtime, and platform.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version.String())
		},
	}
}

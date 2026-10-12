package cmd

import (
	"github.com/spf13/cobra"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/app"
)

// newCitiesCommand builds the cities command.
//
// Returns:
//   - *cobra.Command: the command that lists the predefined city labels.
func newCitiesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cities",
		Short: "List predefined city labels",
		Long:  "Display a list of predefined city labels that can be used for location-based server selection.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return app.ShowCities()
		},
	}
}

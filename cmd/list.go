package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nicholas-fedor/speedtest-go/internal/app"
)

// listCmd represents the list command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available speedtest servers",
	Long:  "Display a list of available speedtest.net servers with their details.",
	RunE: func(_ *cobra.Command, _ []string) error {
		config := app.Config{
			Location:      viper.GetString("location"),
			City:          viper.GetString("city"),
			Search:        viper.GetString("search"),
			Proxy:         viper.GetString("proxy"),
			Source:        viper.GetString("source"),
			DNSBindSource: viper.GetBool("dns-bind-source"),
			UserAgent:     viper.GetString("ua"),
			Debug:         viper.GetBool("debug"),
		}

		return app.RunList(config)
	},
}

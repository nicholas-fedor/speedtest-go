// Package cmd provides CLI commands for the speedtest application.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nicholas-fedor/speedtest-go/internal/app"
	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "speedtest-go",
	Short: "Test internet bandwidth using speedtest.net",
	Long:  "A command-line tool to test internet download and upload speeds using speedtest.net servers.",
	RunE: func(_ *cobra.Command, _ []string) error {
		config := app.Config{
			ServerIDs:     viper.GetIntSlice("server"),
			CustomURL:     viper.GetString("custom-url"),
			SavingMode:    viper.GetBool("saving-mode"),
			JSONOutput:    viper.GetBool("json"),
			JSONLOutput:   viper.GetBool("jsonl"),
			UnixOutput:    viper.GetBool("unix"),
			Proxy:         viper.GetString("proxy"),
			Source:        viper.GetString("source"),
			DNSBindSource: viper.GetBool("dns-bind-source"),
			Multi:         viper.GetBool("multi"),
			Thread:        viper.GetInt("thread"),
			UserAgent:     viper.GetString("ua"),
			NoDownload:    viper.GetBool("no-download"),
			NoUpload:      viper.GetBool("no-upload"),
			PingMode:      viper.GetString("ping-mode"),
			Unit:          viper.GetString("unit"),
			Debug:         viper.GetBool("debug"),
		}

		return app.RunSpeedtest(config)
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
	cobra.OnInitialize(initConfig)

	// Persistent flags (global)
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.speedtest-go.yaml)")
	rootCmd.PersistentFlags().
		String("proxy", "", "Set a proxy(http[s] or socks) for the speedtest.")
	rootCmd.PersistentFlags().String("source", "", "Bind a source interface for the speedtest.")
	rootCmd.PersistentFlags().
		Bool("dns-bind-source", false, "DNS request binding source (experimental).")
	rootCmd.PersistentFlags().String("ua", "", "Set the user-agent header for the speedtest.")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode.")

	// Root command flags (for speedtest)
	rootCmd.Flags().IntSliceP("server", "s", []int{}, "Select server id to run speedtest.")
	rootCmd.Flags().
		String("custom-url", "", "Specify the url of the server instead of fetching from speedtest.net.")
	rootCmd.Flags().
		Bool("saving-mode", false, "Test with few resources, though low accuracy (especially > 30Mbps).")
	rootCmd.Flags().Bool("json", false, "Output results in json format.")
	rootCmd.Flags().
		Bool("jsonl", false, "Output results in jsonl format (one json object per line).")
	rootCmd.Flags().Bool("unix", false, "Output results in unix like format.")
	rootCmd.Flags().BoolP("multi", "m", false, "Enable multi-server mode.")
	rootCmd.Flags().IntP("thread", "t", 0, "Set the number of concurrent connections.")
	rootCmd.Flags().Bool("no-download", false, "Disable download test.")
	rootCmd.Flags().Bool("no-upload", false, "Disable upload test.")
	rootCmd.Flags().String("ping-mode", "http", "Select a method for Ping (support icmp/tcp/http).")
	rootCmd.Flags().
		StringP("unit", "u", "", "Set human-readable and auto-scaled rate units for output "+
			"(options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).")

	// Bind persistent flags to viper
	_ = viper.BindPFlag("proxy", rootCmd.PersistentFlags().Lookup("proxy"))
	_ = viper.BindPFlag("source", rootCmd.PersistentFlags().Lookup("source"))
	_ = viper.BindPFlag("dns-bind-source", rootCmd.PersistentFlags().Lookup("dns-bind-source"))
	_ = viper.BindPFlag("ua", rootCmd.PersistentFlags().Lookup("ua"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Bind root flags to viper
	_ = viper.BindPFlag("server", rootCmd.Flags().Lookup("server"))
	_ = viper.BindPFlag("custom-url", rootCmd.Flags().Lookup("custom-url"))
	_ = viper.BindPFlag("saving-mode", rootCmd.Flags().Lookup("saving-mode"))
	_ = viper.BindPFlag("json", rootCmd.Flags().Lookup("json"))
	_ = viper.BindPFlag("jsonl", rootCmd.Flags().Lookup("jsonl"))
	_ = viper.BindPFlag("unix", rootCmd.Flags().Lookup("unix"))
	_ = viper.BindPFlag("multi", rootCmd.Flags().Lookup("multi"))
	_ = viper.BindPFlag("thread", rootCmd.Flags().Lookup("thread"))
	_ = viper.BindPFlag("no-download", rootCmd.Flags().Lookup("no-download"))
	_ = viper.BindPFlag("no-upload", rootCmd.Flags().Lookup("no-upload"))
	_ = viper.BindPFlag("ping-mode", rootCmd.Flags().Lookup("ping-mode"))
	_ = viper.BindPFlag("unit", rootCmd.Flags().Lookup("unit"))

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(citiesCmd)

	// List command flags
	listCmd.Flags().
		String("location", "", "Change the location with a precise coordinate (format: lat,lon).")
	listCmd.Flags().String("city", "", "Change the location with a predefined city label.")
	listCmd.Flags().String("search", "", "Fuzzy search servers by a keyword.")

	// Bind list flags to viper
	_ = viper.BindPFlag("location", listCmd.Flags().Lookup("location"))
	_ = viper.BindPFlag("city", listCmd.Flags().Lookup("city"))
	_ = viper.BindPFlag("search", listCmd.Flags().Lookup("search"))

	// Set version
	rootCmd.Version = fmt.Sprintf("speedtest-go v%s git-dev built at unknown", speedtest.Version())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".speedtest-go" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".speedtest-go")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

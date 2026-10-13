package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RegisterRootFlags adds persistent and root command flags to rootCmd and binds
// them to viper. cfgFile is bound to the --config persistent flag so InitViper
// can read it.
func RegisterRootFlags(rootCmd *cobra.Command, cfgFile *string) {
	rootCmd.PersistentFlags().
		StringVar(cfgFile, "config", "", "config file (default is $HOME/.speedtest-go.yaml)")
	rootCmd.PersistentFlags().
		String("proxy", "", "Set a proxy(http[s] or socks) for the speedtest.")
	rootCmd.PersistentFlags().
		String("source", "", "Send from a source IP address or network interface name.")
	rootCmd.PersistentFlags().
		Bool("dns-bind-source", false, "DNS request binding source (experimental).")
	rootCmd.PersistentFlags().String("ua", "", "Set the user-agent header for the speedtest.")
	rootCmd.PersistentFlags().
		String("location", "", "Change the location with a precise coordinate (format: lat,lon).")
	rootCmd.PersistentFlags().
		String("city", "", "Change the location with a predefined city label.")
	rootCmd.PersistentFlags().String("search", "", "Fuzzy search servers by a keyword.")
	rootCmd.PersistentFlags().
		StringSlice("filter-cc", nil, "Only use servers in these countries (two-letter codes, or auto for your own).")
	rootCmd.PersistentFlags().
		Int("dscp", 0, "Mark connections with a DSCP code point from 0 to 63 (Linux, macOS and BSD).")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode.")
	rootCmd.PersistentFlags().
		String("base-url", "", "Set the speedtest API base URL (default https://www.speedtest.net).")

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
		String("protocol", "http", "Select the download and upload protocol (http or tcp).")
	rootCmd.Flags().
		Int("dl-size", 0, "Set the download block size, from 350 to 4000 (default 1000).")
	rootCmd.Flags().
		Int("ul-size", 0, "Set the upload block size in kB, from 100 to 4000 (default 1000, or 800 with --multi).")
	rootCmd.Flags().
		StringP("unit", "u", "", "Set human-readable and auto-scaled rate units for output "+
			"(options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).")

	_ = viper.BindPFlag("proxy", rootCmd.PersistentFlags().Lookup("proxy"))
	_ = viper.BindPFlag("source", rootCmd.PersistentFlags().Lookup("source"))
	_ = viper.BindPFlag("dns-bind-source", rootCmd.PersistentFlags().Lookup("dns-bind-source"))
	_ = viper.BindPFlag("ua", rootCmd.PersistentFlags().Lookup("ua"))
	_ = viper.BindPFlag("dscp", rootCmd.PersistentFlags().Lookup("dscp"))
	_ = viper.BindPFlag("filter-cc", rootCmd.PersistentFlags().Lookup("filter-cc"))
	_ = viper.BindPFlag("location", rootCmd.PersistentFlags().Lookup("location"))
	_ = viper.BindPFlag("city", rootCmd.PersistentFlags().Lookup("city"))
	_ = viper.BindPFlag("search", rootCmd.PersistentFlags().Lookup("search"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("base-url", rootCmd.PersistentFlags().Lookup("base-url"))

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
	_ = viper.BindPFlag("protocol", rootCmd.Flags().Lookup("protocol"))
	_ = viper.BindPFlag("dl-size", rootCmd.Flags().Lookup("dl-size"))
	_ = viper.BindPFlag("ul-size", rootCmd.Flags().Lookup("ul-size"))
	_ = viper.BindPFlag("unit", rootCmd.Flags().Lookup("unit"))
}

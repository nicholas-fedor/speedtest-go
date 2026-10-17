package flags

import (
	"github.com/spf13/cobra"
)

// Names of the flags that every command takes.
const (
	// Config names the settings file.
	Config = "config"
	// Proxy sets a proxy for the speed test.
	Proxy = "proxy"
	// Source sets the IP address or network interface the connections leave from.
	Source = "source"
	// DNSBindSource sends DNS requests from the source too.
	DNSBindSource = "dns-bind-source"
	// UserAgent sets the user-agent header.
	UserAgent = "ua"
	// Location chooses servers near coordinates.
	Location = "location"
	// City chooses servers near a predefined city.
	City = "city"
	// Search narrows the servers by a keyword.
	Search = "search"
	// FilterCC keeps only the servers in some countries.
	FilterCC = "filter-cc"
	// DSCP marks the connections with a code point.
	DSCP = "dscp"
	// MaskISP masks the user's address in all output.
	MaskISP = "mask-isp"
	// Debug turns on the debug messages.
	Debug = "debug"
	// BaseURL sets the speedtest API base URL.
	BaseURL = "base-url"
)

// Names of the flags that only the speed test takes.
const (
	// Server selects servers by ID.
	Server = "server"
	// CustomURL tests a server that speedtest.net does not list.
	CustomURL = "custom-url"
	// SavingMode tests with few resources.
	SavingMode = "saving-mode"
	// JSON writes the results as one JSON document.
	JSON = "json"
	// JSONL writes the results as one JSON line per server.
	JSONL = "jsonl"
	// Unix writes the results as plain lines.
	Unix = "unix"
	// Multi spreads each transfer test across several servers.
	Multi = "multi"
	// Thread sets the number of connections.
	Thread = "thread"
	// NoDownload skips the download test.
	NoDownload = "no-download"
	// NoUpload skips the upload test.
	NoUpload = "no-upload"
	// PingMode selects how latency is measured.
	PingMode = "ping-mode"
	// Protocol selects the download and upload protocol.
	Protocol = "protocol"
	// DownloadSize sets the download block size.
	DownloadSize = "dl-size"
	// UploadSize sets the upload block size.
	UploadSize = "ul-size"
	// Unit sets the unit of the rates.
	Unit = "unit"
)

// RegisterRootFlags adds the flags that every command takes as persistent flags of rootCmd, and the flags that only
// the speed test takes as its local flags.
//
// Parameters:
//   - rootCmd: the root command.
func RegisterRootFlags(rootCmd *cobra.Command) {
	persistent := rootCmd.PersistentFlags()
	persistent.String(
		Config,
		"",
		"settings file (default speedtest-go/config.yaml in the user config directory)",
	)
	persistent.String(Proxy, "", "Set a proxy(http[s] or socks) for the speedtest.")
	persistent.String(Source, "", "Send from a source IP address or network interface name.")
	persistent.Bool(DNSBindSource, false, "DNS request binding source (experimental).")
	persistent.String(UserAgent, "", "Set the user-agent header for the speedtest.")
	persistent.String(
		Location,
		"",
		"Change the location with a precise coordinate (format: lat,lon).",
	)
	persistent.String(City, "", "Change the location with a predefined city label.")
	persistent.String(Search, "", "Fuzzy search servers by a keyword.")
	persistent.StringSlice(
		FilterCC,
		nil,
		"Only use servers in these countries (two-letter codes, or auto for your own).",
	)
	persistent.Int(
		DSCP,
		0,
		"Mark connections with a DSCP code point from 0 to 63 (Linux, macOS and BSD).",
	)
	persistent.Bool(
		MaskISP,
		false,
		"Mask your IP address, coordinates, and distances to servers in all output.",
	)
	persistent.Bool(Debug, false, "Enable debug mode.")
	persistent.String(
		BaseURL,
		"",
		"Set the speedtest API base URL (default https://www.speedtest.net).",
	)

	local := rootCmd.Flags()
	local.IntSliceP(Server, "s", []int{}, "Select server id to run speedtest.")
	local.String(
		CustomURL,
		"",
		"Specify the url of the server instead of fetching from speedtest.net.",
	)
	local.Bool(
		SavingMode,
		false,
		"Test with few resources, though low accuracy (especially > 30Mbps).",
	)
	local.Bool(JSON, false, "Output results in json format.")
	local.Bool(JSONL, false, "Output results in jsonl format (one json object per line).")
	local.Bool(Unix, false, "Output results in unix like format.")
	local.BoolP(Multi, "m", false, "Enable multi-server mode.")
	local.IntP(Thread, "t", 0, "Set the number of concurrent connections.")
	local.Bool(NoDownload, false, "Disable download test.")
	local.Bool(NoUpload, false, "Disable upload test.")
	local.String(PingMode, "http", "Select a method for Ping (support icmp/tcp/http).")
	local.String(Protocol, "http", "Select the download and upload protocol (http or tcp).")
	local.Int(DownloadSize, 0, "Set the download block size, from 350 to 4000 (default 1000).")
	local.Int(
		UploadSize,
		0,
		"Set the upload block size in kB, from 100 to 4000 (default 1000, or 800 with --multi).",
	)
	local.StringP(Unit, "u", "", "Set human-readable and auto-scaled rate units for output "+
		"(options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).")
}

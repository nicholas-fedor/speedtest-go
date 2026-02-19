// Package output provides functions for displaying speed test results and application information.
package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

var (
	commit = "dev"
	date   = "unknown"
)

// ShowServerList prints the list of available servers.
func ShowServerList(servers speedtest.Servers) {
	for _, server := range servers {
		_, _ = fmt.Fprintf(os.Stdout, "[%5s] %9.2fkm ", server.ID, server.Distance)

		if server.Latency == -1 {
			_, _ = fmt.Fprintf(os.Stdout, "%v", "Timeout ")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "%-dms ", server.Latency/time.Millisecond)
		}

		_, _ = fmt.Fprintf(
			os.Stdout,
			"\t%s (%s) by %s \n",
			server.Name,
			server.Country,
			server.Sponsor,
		)
	}
}

// AppInfo prints application information.
func AppInfo(jsonOutput, jsonlOutput bool) {
	if !jsonOutput && !jsonlOutput {
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintf(
			os.Stdout,
			"    speedtest-go v%s (git-%s, %s) @nicholas-fedor\n",
			speedtest.Version(),
			commit,
			date,
		)
		_, _ = fmt.Fprintln(os.Stdout)
	}
}

// Version returns a formatted version string.
func Version() string {
	version := speedtest.Version()
	// Only add 'v' prefix for proper version numbers (containing a dot)
	if strings.Contains(version, ".") && !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	// Shorten commit hash to 7 characters
	commit := commit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	// Format date as YYYY-MM-DD
	buildDate := date
	if len(buildDate) > 10 {
		buildDate = buildDate[:10]
	}

	return fmt.Sprintf("%s %s %s", version, commit, buildDate)
}

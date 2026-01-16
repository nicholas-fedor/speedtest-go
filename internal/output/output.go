// Package output provides functions for displaying speed test results and application information.
package output

import (
	"fmt"
	"os"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

var commit = "dev"

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
			"    speedtest-go v%s (git-%s) @showwin\n",
			speedtest.Version(),
			commit,
		)
		_, _ = fmt.Fprintln(os.Stdout)
	}
}

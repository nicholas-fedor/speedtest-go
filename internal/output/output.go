// Package output provides functions for displaying speed test results and application information.
package output

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// ShowServerList prints the list of available servers.
func ShowServerList(servers speedtest.Servers) {
	writeServerList(os.Stdout, servers, formatDistance)
}

// ShowMaskedServerList prints the list of available servers for --mask-isp, with each distance masked.
//
// Parameters:
//   - servers: the servers to list.
func ShowMaskedServerList(servers speedtest.Servers) {
	writeServerList(
		os.Stdout,
		servers,
		func(distance float64) string { return maskValue(formatDistance(distance)) },
	)
}

// writeServerList writes one line per server.
//
// Parameters:
//   - writer: where to write.
//   - servers: the servers to list.
//   - distance: formats a server's distance from the user.
func writeServerList(writer io.Writer, servers speedtest.Servers, distance func(float64) string) {
	for _, server := range servers {
		_, _ = fmt.Fprintf(writer, "[%5s] %skm ", server.ID, distance(server.Distance))

		if server.Latency == -1 {
			_, _ = fmt.Fprintf(writer, "%v", "Timeout ")
		} else {
			_, _ = fmt.Fprintf(writer, "%-dms ", server.Latency/time.Millisecond)
		}

		_, _ = fmt.Fprintf(
			writer,
			"\t%s (%s) by %s \n",
			server.Name,
			server.Country,
			server.Sponsor,
		)
	}
}

// formatDistance formats a distance in kilometres for the server list column.
//
// Parameters:
//   - distance: the distance in kilometres.
//
// Returns:
//   - string: the distance padded to the column width, without a unit.
func formatDistance(distance float64) string {
	return fmt.Sprintf("%9.2f", distance)
}

// AppInfo prints application information.
func AppInfo(jsonOutput, jsonlOutput bool) {
	if jsonOutput || jsonlOutput {
		return
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "    %s\n", version.Banner())
	_, _ = fmt.Fprintln(os.Stdout)
}

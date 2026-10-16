// Package output provides functions for displaying speed test results and application information.
package output

import (
	"fmt"
	"io"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/version"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// WriteServerList writes the list of available servers, one line per server.
//
// Parameters:
//   - writer: where to write.
//   - servers: the servers to list.
func WriteServerList(writer io.Writer, servers speedtest.Servers) {
	writeServerList(writer, servers, formatDistance)
}

// WriteMaskedServerList writes the list of available servers for --mask-isp, with each distance masked.
//
// Parameters:
//   - writer: where to write.
//   - servers: the servers to list.
func WriteMaskedServerList(writer io.Writer, servers speedtest.Servers) {
	writeServerList(
		writer,
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

// WriteAppInfo writes the banner with the application's version, between blank lines.
//
// Parameters:
//   - writer: where to write.
func WriteAppInfo(writer io.Writer) {
	_, _ = fmt.Fprintf(writer, "\n    %s\n\n", version.Banner())
}

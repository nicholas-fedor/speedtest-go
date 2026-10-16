// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Options choose how a reporter formats its lines.
type Options struct {
	// Unit is the unit of the rates.
	Unit speedtest.UnitType
	// MaskISP masks the user's address, coordinates, and the distances to servers.
	MaskISP bool
}

// Fixed lines and spinner titles.
const (
	// userTitle is shown while the user's details are looked up.
	userTitle = "Retrieving User Information"
	// serversTitle is shown while the servers are looked up.
	serversTitle = "Retrieving Servers"
	// pingTitle is shown before the first round trip of the latency measurement.
	pingTitle = "Latency: --"
	// packetLossStartedLine reports the background packet loss measurement.
	packetLossStartedLine = "Packet Loss Analyzer: Running in background (<= 30 Secs)"
)

// Unit conversions for the transfer lines.
const (
	// bytesPerMB converts bytes to the megabytes of the Used figure.
	bytesPerMB = 1000 * 1000
	// percent converts a ratio to a percentage.
	percent = 100
)

// userLine describes the user's connection.
//
// Parameters:
//   - user: the user's details.
//   - options: whether to mask them.
//
// Returns:
//   - string: the ISP line.
func userLine(user *speedtest.User, options Options) string {
	if options.MaskISP {
		user = output.MaskUser(user)
	}

	return "ISP: " + user.String()
}

// serversLine describes the servers a run found.
//
// Parameters:
//   - found: the servers found.
//
// Returns:
//   - string: the line.
func serversLine(found runner.ServersFound) string {
	switch {
	case found.Source == runner.SourceCustom:
		return "Skip: Using Custom Server"
	case found.Source == runner.SourceSpecified:
		return fmt.Sprintf("Found %d Specified Public Server(s)", found.Count)
	case len(found.Codes) > 0:
		return fmt.Sprintf(
			"Found %d Public Servers in %s",
			found.Count,
			strings.Join(found.Codes, ", "),
		)
	default:
		return fmt.Sprintf("Found %d Public Servers", found.Count)
	}
}

// serverLine names the server being tested.
//
// Parameters:
//   - server: the server.
//   - options: whether to mask its distance.
//
// Returns:
//   - string: the Test Server line.
func serverLine(server *speedtest.Server, options Options) string {
	if options.MaskISP {
		return "Test Server: " + output.MaskedServerLine(server)
	}

	return "Test Server: " + server.String()
}

// pingLine describes a server's latency.
//
// Parameters:
//   - stats: the measured round trips.
//
// Returns:
//   - string: the Latency line.
func pingLine(stats speedtest.PingStats) string {
	return fmt.Sprintf(
		"Latency: %v Jitter: %v Min: %v Max: %v",
		stats.Latency,
		stats.Jitter,
		stats.Min,
		stats.Max,
	)
}

// progressLine describes a transfer test while it runs.
//
// Parameters:
//   - progress: the live rate and latency.
//   - options: the rate unit.
//
// Returns:
//   - string: the progress line.
func progressLine(progress runner.TransferProgress, options Options) string {
	if progress.Latency == 0 {
		return fmt.Sprintf(
			"%s: %s (Latency: --)",
			progress.Direction,
			progress.Rate.Format(options.Unit),
		)
	}

	return fmt.Sprintf(
		"%s: %s (Latency: %dms)",
		progress.Direction,
		progress.Rate.Format(options.Unit),
		progress.Latency.Milliseconds(),
	)
}

// transferLine describes a finished transfer test with the latency under its load.
//
// Parameters:
//   - finished: the transfer and its loaded latency.
//   - options: the rate unit.
//
// Returns:
//   - string: the Download or Upload line.
func transferLine(finished runner.TransferFinished, options Options) string {
	transfer := finished.Transfer

	used := fmt.Sprintf("Used: %.2fMB", float64(transfer.Bytes)/bytesPerMB)
	if finished.Direction == runner.Upload {
		used = fmt.Sprintf(
			"Used: %.2fMB Confirmed: %.1f%%",
			float64(transfer.Bytes)/bytesPerMB,
			transfer.ConfirmationRatio()*percent,
		)
	}

	return fmt.Sprintf(
		"%s: %s (%s) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)",
		finished.Direction,
		transfer.Rate.Format(options.Unit),
		used,
		finished.Latency.Latency.Milliseconds(),
		finished.Latency.Jitter.Milliseconds(),
		finished.Latency.Min.Milliseconds(),
		finished.Latency.Max.Milliseconds(),
	)
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Layout of a summary panel.
const (
	// panelLabelWidth is the width of the labels in a summary panel.
	panelLabelWidth = 12
	// panelValueWidth is the width of the values in a summary panel.
	panelValueWidth = 13
	// panelGap separates the two columns of a summary panel.
	panelGap = "   "
	// notMeasured stands for a test that the settings skipped or that measured nothing.
	notMeasured = "N/A"
)

// panelBorder draws the frame of a summary panel.
var panelBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

// summaryPanel draws the headline results of a server in a framed panel, with the server in the frame.
//
// Parameters:
//   - result: the server's measurements.
//   - options: the rate unit.
//
// Returns:
//   - string: the panel's lines.
func summaryPanel(result speedtest.Result, options Options) string {
	body := strings.Join([]string{
		panelRow(
			"Download",
			transferRate(result.Download, options),
			"Upload",
			transferRate(result.Upload, options),
		),
		panelRow("Latency", pingLatency(result.Ping), "Packet loss", packetLoss(result.PacketLoss)),
	}, "\n")

	lines := strings.Split(panelBorder.Render(body), "\n")
	lines[0] = titledBorder(lines[0], serverTitle(result.Server))

	return strings.Join(lines, "\n")
}

// panelRow lays out two labeled values in the columns of a summary panel.
//
// Parameters:
//   - leftLabel: the first label.
//   - leftValue: the first value.
//   - rightLabel: the second label.
//   - rightValue: the second value.
//
// Returns:
//   - string: the row.
func panelRow(leftLabel, leftValue, rightLabel, rightValue string) string {
	return panelCell(leftLabel, leftValue) + panelGap + panelCell(rightLabel, rightValue)
}

// panelCell lays out a label and its value.
//
// Parameters:
//   - label: the label.
//   - value: the value.
//
// Returns:
//   - string: the label, dimmed, and the value, highlighted, each padded to its column.
func panelCell(label, value string) string {
	return faintStyle.Render(fmt.Sprintf("%-*s", panelLabelWidth, label)) +
		rateStyle.Render(fmt.Sprintf("%-*s", panelValueWidth, value))
}

// titledBorder puts a title into the top line of a frame, after its corner.
//
// Parameters:
//   - top: the frame's top line.
//   - title: the title.
//
// Returns:
//   - string: the top line with the title, cut to fit when it is too long.
func titledBorder(top, title string) string {
	// The title takes the frame's top line between a rule before it and at least one rule after it, with a space on
	// each side.
	const around = 4

	inner := ansi.StringWidth(top) - 2 //nolint:mnd // The two corners.
	room := inner - around

	if room <= 0 || title == "" {
		return top
	}

	title = ansi.Truncate(title, room, "…")
	border := lipgloss.RoundedBorder()

	return border.TopLeft + border.Top + " " + titleStyle.Render(title) + " " +
		strings.Repeat(border.Top, inner-around+1-ansi.StringWidth(title)) + border.TopRight
}

// serverTitle names a server by its sponsor and city.
//
// Parameters:
//   - server: the server, or nil.
//
// Returns:
//   - string: the title, or empty without a server.
func serverTitle(server *speedtest.Server) string {
	if server == nil {
		return ""
	}

	return server.Sponsor + " · " + server.Name
}

// transferRate formats the rate of a transfer test.
//
// Parameters:
//   - transfer: the test, or nil when it was skipped.
//   - options: the rate unit.
//
// Returns:
//   - string: the rate, or N/A.
func transferRate(transfer *speedtest.Transfer, options Options) string {
	if transfer == nil {
		return notMeasured
	}

	return transfer.Rate.Format(options.Unit)
}

// pingLatency formats the mean latency of a server.
//
// Parameters:
//   - ping: the latency, or nil.
//
// Returns:
//   - string: the latency in milliseconds, or N/A.
func pingLatency(ping *speedtest.PingStats) string {
	if ping == nil {
		return notMeasured
	}

	return fmt.Sprintf(
		"%.2f ms",
		float64(ping.Latency.Microseconds())/1000,
	) //nolint:mnd // Microseconds to ms.
}

// packetLoss formats the packet loss of a server.
//
// Parameters:
//   - loss: the packet loss, or nil when nothing was measured.
//
// Returns:
//   - string: the loss as a percentage, or N/A.
func packetLoss(loss *speedtest.PacketLossStats) string {
	if loss == nil {
		return notMeasured
	}

	return fmt.Sprintf("%.2f%%", loss.LossPercent())
}

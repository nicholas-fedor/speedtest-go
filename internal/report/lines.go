// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"io"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
)

// Lines writes one line for each result of a run, without live progress, for --unix output.
type Lines struct {
	// out receives the lines.
	out io.Writer
	// options format the lines.
	options Options
}

// Compile-time check that [Lines] is a [runner.Reporter].
var _ runner.Reporter = (*Lines)(nil)

// NewLines creates a line reporter.
//
// Parameters:
//   - out: receives the lines.
//   - options: format the lines.
//
// Returns:
//   - *Lines: the reporter.
func NewLines(out io.Writer, options Options) *Lines {
	return &Lines{out: out, options: options}
}

// Report writes the line of an event that has a result. Progress events have none.
//
// Parameters:
//   - event: the event.
func (l *Lines) Report(event runner.Event) {
	switch reported := event.(type) {
	case runner.UserFound:
		l.println(userLine(reported.User, l.options))
	case runner.ServersFound:
		l.println(serversLine(reported))
	case runner.ServerStarted:
		l.println("")
		l.println(serverLine(reported.Server, l.options))
	case runner.PingFinished:
		l.println(pingLine(reported.Stats))
	case runner.PacketLossStarted:
		l.println(packetLossStartedLine)
	case runner.TransferFinished:
		l.println(transferLine(reported, l.options))
	case runner.PacketLossFinished:
		l.println(reported.Stats.String())
	default:
		// Progress events change nothing that a finished line shows.
	}
}

// println writes one line. Progress output is best effort, so a write error is ignored.
//
// Parameters:
//   - line: the line, without its newline.
func (l *Lines) println(line string) {
	_, _ = fmt.Fprintln(l.out, line)
}

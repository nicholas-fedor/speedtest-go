// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"io"
	"slices"

	"github.com/chelnak/ysmrr"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
)

// Spinner shows the progress of a run with a spinner for each lookup and test, which turns into its result line.
type Spinner struct {
	// manager draws the spinners of the current server.
	manager ysmrr.SpinnerManager
	// out receives the spinners and the blank line before each server.
	out io.Writer
	// user is the spinner of the user lookup.
	user *ysmrr.Spinner
	// servers is the spinner of the server lookup.
	servers *ysmrr.Spinner
	// current is the spinner of the running test.
	current *ysmrr.Spinner
	// active are the spinners that have not finished.
	active []*ysmrr.Spinner
	// options format the lines.
	options Options
}

// Compile-time check that [Spinner] is a [runner.Reporter].
var _ runner.Reporter = (*Spinner)(nil)

// NewSpinner creates a spinner reporter and starts drawing.
//
// Parameters:
//   - out: receives the spinners.
//   - options: format the lines.
//
// Returns:
//   - *Spinner: the reporter, which stops drawing when the run finishes.
func NewSpinner(out io.Writer, options Options) *Spinner {
	reporter := &Spinner{
		manager: newSpinnerManager(out),
		out:     out,
		user:    nil,
		servers: nil,
		current: nil,
		active:  nil,
		options: options,
	}

	reporter.manager.Start()

	return reporter
}

// Report shows an event: it starts a spinner when a lookup or test starts, updates it with progress, and turns it
// into its result line when the lookup or test finishes. Each server starts a new group of spinners.
//
// Parameters:
//   - event: the event.
func (s *Spinner) Report(event runner.Event) {
	switch reported := event.(type) {
	case runner.LookupStarted:
		s.user = s.start(userTitle)
		s.servers = s.start(serversTitle)
	case runner.UserFound:
		s.finish(s.user, userLine(reported.User, s.options))
	case runner.ServersFound:
		s.finish(s.servers, serversLine(reported))
	case runner.ServerStarted:
		s.reset()
		_, _ = fmt.Fprintln(s.out)
		s.line(serverLine(reported.Server, s.options))
	case runner.PingStarted:
		s.current = s.start(pingTitle)
	case runner.LatencyMeasured:
		s.update(fmt.Sprintf("Latency: %v", reported.Latency))
	case runner.PingFinished:
		s.finish(s.current, pingLine(reported.Stats))
	case runner.PacketLossStarted:
		s.line(packetLossStartedLine)
	case runner.TransferStarted:
		s.current = s.start(reported.Direction.String())
	case runner.TransferProgress:
		s.update(progressLine(reported, s.options))
	case runner.TransferFinished:
		s.finish(s.current, transferLine(reported, s.options))
	case runner.PacketLossFinished:
		s.line(reported.Stats.String())
	case runner.RunFinished:
		s.stop(reported.Err)
	default:
		// The remaining events change nothing that a spinner shows.
	}
}

// start adds a running spinner.
//
// Parameters:
//   - title: the spinner's message.
//
// Returns:
//   - *ysmrr.Spinner: the spinner.
func (s *Spinner) start(title string) *ysmrr.Spinner {
	spinner := s.manager.AddSpinner(title)
	s.active = append(s.active, spinner)

	return spinner
}

// update changes the message of the running test's spinner.
//
// Parameters:
//   - message: the new message.
func (s *Spinner) update(message string) {
	if s.current != nil {
		s.current.UpdateMessage(message)
	}
}

// finish turns a spinner into its result line.
//
// Parameters:
//   - spinner: the spinner, or nil when none was started.
//   - message: the result line.
func (s *Spinner) finish(spinner *ysmrr.Spinner, message string) {
	if spinner == nil {
		return
	}

	spinner.UpdateMessage(message)
	spinner.Complete()

	s.active = slices.DeleteFunc(
		s.active,
		func(active *ysmrr.Spinner) bool { return active == spinner },
	)
}

// line adds a finished line.
//
// Parameters:
//   - message: the line.
func (s *Spinner) line(message string) {
	s.finish(s.start(message), message)
}

// reset stops drawing the finished spinners and starts a new group.
func (s *Spinner) reset() {
	s.manager.Stop()
	s.manager = newSpinnerManager(s.out)
	s.manager.Start()

	s.user, s.servers, s.current = nil, nil, nil
	s.active = nil
}

// stop marks the unfinished spinners as failed when the run failed, then stops drawing.
//
// Parameters:
//   - err: the error that ended the run, or nil.
func (s *Spinner) stop(err error) {
	if err != nil {
		for _, spinner := range s.active {
			spinner.Error()
		}
	}

	s.active = nil
	s.manager.Stop()
}

// newSpinnerManager creates a spinner manager that draws to out.
//
// Parameters:
//   - out: receives the spinners.
//
// Returns:
//   - ysmrr.SpinnerManager: the manager, not yet started.
func newSpinnerManager(out io.Writer) ysmrr.SpinnerManager {
	return ysmrr.NewSpinnerManager(ysmrr.WithWriter(out))
}

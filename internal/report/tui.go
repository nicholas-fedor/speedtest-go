// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
)

// TUI shows the progress of a run in the terminal: a spinner row for each running lookup or test, and a line for
// each result printed above the rows.
//
// It draws inline, never on the alternate screen, so the results stay in the terminal's scrollback. It reads no
// input, so Ctrl-C reaches the CLI's interrupt handling.
type TUI struct {
	// program draws the rows.
	program *tea.Program
	// err is the error that ended the program, set before done is closed.
	err error
	// done is closed when the program has ended.
	done chan struct{}
	// options format the lines.
	options Options
}

// Compile-time check that [TUI] is a [runner.Reporter].
var _ runner.Reporter = (*TUI)(nil)

// NewTUI creates a terminal reporter and starts drawing.
//
// Parameters:
//   - out: the terminal.
//   - options: format the lines.
//   - programOptions: further program options, such as a fixed window size for tests.
//
// Returns:
//   - *TUI: the reporter, which stops drawing when the run finishes.
func NewTUI(out io.Writer, options Options, programOptions ...tea.ProgramOption) *TUI {
	program := tea.NewProgram(newModel(), append([]tea.ProgramOption{
		tea.WithInput(nil),
		tea.WithOutput(out),
		tea.WithoutSignalHandler(),
	}, programOptions...)...)

	reporter := &TUI{program: program, err: nil, done: make(chan struct{}), options: options}

	go func() {
		defer close(reporter.done)

		_, reporter.err = program.Run()
	}()

	return reporter
}

// Report shows an event: a lookup or test that starts gets a row, its progress updates the row, and its result
// replaces the row with a finished line. [runner.RunFinished] waits until everything is drawn.
//
// Parameters:
//   - event: the event.
func (t *TUI) Report(event runner.Event) {
	switch reported := event.(type) {
	case runner.LookupStarted:
		t.send(startRow{text: userTitle, key: userRow})
		t.send(startRow{text: serversTitle, key: serversRow})
	case runner.UserFound:
		t.finish(userRow, userLine(reported.User, t.options))
	case runner.ServersFound:
		t.finish(serversRow, serversLine(reported))
	case runner.ServerStarted:
		// An empty print is dropped, so the blank line before each server goes out with the server's line.
		t.send(tea.Println("\n" + doneMark + " " + serverLine(reported.Server, t.options))())
	case runner.PingStarted:
		t.send(startRow{text: pingTitle, key: testRow})
	case runner.LatencyMeasured:
		t.send(updateRow{text: fmt.Sprintf("Latency: %v", reported.Latency), key: testRow})
	case runner.PingFinished:
		t.finish(testRow, pingLine(reported.Stats))
	case runner.PacketLossStarted:
		t.print(packetLossStartedLine)
	case runner.TransferStarted:
		t.send(startRow{text: reported.Direction.String(), key: testRow})
	case runner.TransferProgress:
		t.send(updateRow{text: progressLine(reported, t.options), key: testRow})
	case runner.TransferFinished:
		t.finish(testRow, transferLine(reported, t.options))
	case runner.PacketLossFinished:
		t.print(reported.Stats.String())
	case runner.RunFinished:
		t.send(endRun{failed: reported.Err != nil})
		<-t.done
	default:
		// The remaining events change nothing the terminal shows.
	}
}

// Err returns the error that ended the drawing. A program that failed drew nothing after it ended, so a run whose
// reporter failed lost results that it could not show.
//
// Returns:
//   - error: the program's error once it has ended, or nil while it runs and after it ended cleanly.
func (t *TUI) Err() error {
	select {
	case <-t.done:
		if t.err != nil {
			return fmt.Errorf("failed to draw the progress: %w", t.err)
		}

		return nil
	default:
		return nil
	}
}

// finish replaces a row with its finished line.
//
// Parameters:
//   - key: the row.
//   - line: the finished line.
func (t *TUI) finish(key rowKey, line string) {
	t.send(endRow{key: key})
	t.print(line)
}

// print prints a finished line above the rows.
//
// Parameters:
//   - line: the line, without its mark.
func (t *TUI) print(line string) {
	// The print goes through the program's message queue, not a command, so lines keep the order of the events.
	t.send(tea.Println(doneMark + " " + line)())
}

// send passes a message to the program. Once the program has ended, the message is dropped.
//
// Parameters:
//   - msg: the message.
func (t *TUI) send(msg tea.Msg) {
	select {
	case <-t.done:
	default:
		t.program.Send(msg)
	}
}

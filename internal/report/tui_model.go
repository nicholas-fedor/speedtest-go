// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// model is the bubbletea model of the [TUI] reporter. It draws a spinner row for each running lookup or test, and a
// running transfer test's row ends with a bar of its progress and its estimated time left. Finished lines are printed
// above the rows, so they stay in the terminal's scrollback.
type model struct {
	// transfer is the running transfer test, or nil.
	transfer *transferView
	// width is shared with the reporter, which sizes the summary panels by it. It is 0 while the width is unknown.
	width *atomic.Int64
	// rows are the running lookups and tests other than transfers, in the order they started.
	rows []row
	// spinner animates the rows.
	spinner spinner.Model
	// bar draws the progress of a transfer test.
	bar progress.Model
	// options format the rates.
	options Options
	// quitting is set when the run finished, so the last frame leaves nothing behind.
	quitting bool
}

// row is a running lookup or test.
type row struct {
	// text describes it.
	text string
	// key identifies it.
	key rowKey
}

// transferView is the state of a running transfer test.
type transferView struct {
	// remaining is the estimated time left.
	remaining time.Duration
	// done is the share of the test done so far, from 0 to 1. It never shrinks, so the bar never moves back when the
	// estimate of the time left grows.
	done float64
	// latency is the latest round trip under the load, or 0 before the first.
	latency time.Duration
	// rate is the latest live rate.
	rate speedtest.ByteRate
	// direction is whether the test downloads or uploads.
	direction runner.Direction
	// measured is set once the first rate arrived.
	measured bool
}

// rowKey identifies a row, so its updates and its end find it.
type rowKey int

// Row keys. The lookups of the caller and the servers run at the same time, and then one test runs at a time.
const (
	// userRow is the lookup of the caller.
	userRow rowKey = iota
	// serversRow is the lookup of the servers.
	serversRow
	// testRow is the running latency test.
	testRow
)

// Layout of a running transfer test's row and of the summary panels.
const (
	// panelWidth is the narrowest terminal that shows the summary panels.
	panelWidth = 50
	// barWidth is the width of the progress bar of a transfer test.
	barWidth = 20
	// barGap separates the bar from the row's text and from the time left.
	barGap = "  "
)

// Messages the [TUI] reporter sends to its model.
type (
	// startRow adds a row.
	startRow row
	// updateRow changes the text of a row.
	updateRow row
	// endRow removes a row once its finished line is printed.
	endRow struct {
		// key identifies the row.
		key rowKey
	}
	// startTransfer shows a transfer test that started.
	startTransfer struct {
		// direction is whether it downloads or uploads.
		direction runner.Direction
	}
	// updateTransfer records the live state of the running transfer test.
	updateTransfer struct {
		// rate is the live rate.
		rate speedtest.ByteRate
		// latency is the latest round trip under the load, or 0 before the first.
		latency time.Duration
		// elapsed is the time since the test started measuring.
		elapsed time.Duration
		// remaining is the estimated time left.
		remaining time.Duration
	}
	// endTransfer removes the transfer test once its finished line is printed.
	endTransfer struct{}
	// endRun ends the program, printing what never finished as failed when the run failed.
	endRun struct {
		// failed is set when the run failed.
		failed bool
	}
)

// Styles of the marks, the rows, and the summary panels.
var (
	// doneMark marks a finished line.
	doneMark = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
	// failedMark marks a lookup or test that a failed run left unfinished.
	failedMark = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗")
	// spinnerStyle colors the spinner of a running row.
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	// titleStyle marks the title of a summary panel.
	titleStyle = lipgloss.NewStyle().Bold(true)
	// rateStyle highlights a value of a summary panel.
	rateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	// faintStyle dims the secondary details.
	faintStyle = lipgloss.NewStyle().Faint(true)
)

// newModel creates the model with nothing running.
//
// Parameters:
//   - options: format the rates.
//   - width: receives the terminal width for the reporter.
//
// Returns:
//   - model: the model.
func newModel(options Options, width *atomic.Int64) model {
	return model{
		transfer: nil,
		width:    width,
		rows:     nil,
		spinner: spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(spinnerStyle),
		),
		bar: progress.New(
			progress.WithFillCharacters('━', '─'),
			progress.WithColors(lipgloss.Color("6")),
			progress.WithoutPercentage(),
			progress.WithWidth(barWidth),
		),
		options:  options,
		quitting: false,
	}
}

// Init starts the spinner.
//
// Returns:
//   - tea.Cmd: the spinner's first tick.
func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update applies a message: a row or transfer change, the end of the run, a terminal resize, or a spinner tick.
//
// Parameters:
//   - msg: the message.
//
// Returns:
//   - tea.Model: the updated model.
//   - tea.Cmd: the spinner's next tick, the prints and quit at the end of the run, or nil.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startRow:
		m.rows = append(slices.Clone(m.rows), row(msg))
	case updateRow:
		m.rows = slices.Clone(m.rows)

		for index := range m.rows {
			if m.rows[index].key == msg.key {
				m.rows[index].text = msg.text
			}
		}
	case endRow:
		m.rows = slices.DeleteFunc(
			slices.Clone(m.rows),
			func(running row) bool { return running.key == msg.key },
		)
	case startTransfer:
		m.transfer = &transferView{
			remaining: 0,
			done:      0,
			latency:   0,
			rate:      0,
			direction: msg.direction,
			measured:  false,
		}
	case updateTransfer:
		m.transfer = m.transfer.with(msg)
	case endTransfer:
		m.transfer = nil
	case endRun:
		return m.endRun(msg.failed)
	case tea.WindowSizeMsg:
		m.width.Store(int64(msg.Width))
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	default:
		// The program's other messages change nothing the view shows.
	}

	return m, nil
}

// View draws one line per running row, each cut to the terminal width.
//
// Returns:
//   - tea.View: the running lookups and tests, or nothing once the run finished.
func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	lines := make([]string, 0, len(m.rows)+1)

	for _, running := range m.rows {
		lines = append(lines, m.fit(m.spinner.View()+" "+running.text))
	}

	if m.transfer != nil {
		lines = append(lines, m.transferRow())
	}

	return tea.NewView(strings.Join(lines, "\n"))
}

// transferRow draws the running transfer test as a row: its progress line, then a bar of the share done and the
// estimated time left. The bar is left out until the first estimate and when the terminal is too narrow for it.
//
// Returns:
//   - string: the row.
func (m model) transferRow() string {
	transfer := m.transfer

	row := m.spinner.View() + " " + transfer.direction.String()
	if !transfer.measured {
		return m.fit(row)
	}

	row = m.spinner.View() + " " + progressLine(transfer.progress(), m.options)
	withBar := row + barGap + m.bar.ViewAs(
		transfer.done,
	) + barGap + faintStyle.Render(
		timeLeft(transfer.remaining),
	)

	if width := int(m.width.Load()); width > 0 && ansi.StringWidth(withBar) > width {
		return m.fit(row)
	}

	return withBar
}

// endRun quits the program. When the run failed, the rows and the transfer test that never finished are printed as
// failed first.
//
// Parameters:
//   - failed: whether the run failed.
//
// Returns:
//   - tea.Model: the model, which draws nothing more.
//   - tea.Cmd: the prints, in order, and then quit.
func (m model) endRun(failed bool) (tea.Model, tea.Cmd) {
	m.quitting = true

	cmds := make(
		[]tea.Cmd,
		0,
		len(m.rows)+2,
	) //nolint:mnd // A transfer test and quit follow the rows.

	if failed {
		for _, unfinished := range m.rows {
			cmds = append(cmds, tea.Println(failedMark+" "+unfinished.text))
		}

		if m.transfer != nil {
			cmds = append(
				cmds,
				tea.Println(failedMark+" "+progressLine(m.transfer.progress(), m.options)),
			)
		}
	}

	m.rows, m.transfer = nil, nil

	return m, tea.Sequence(append(cmds, tea.Quit)...)
}

// fit cuts a line to the terminal width, so a long line never wraps and breaks the redraw.
//
// Parameters:
//   - line: the styled line.
//
// Returns:
//   - string: the line, cut with an ellipsis when it is wider than the terminal.
func (m model) fit(line string) string {
	width := int(m.width.Load())
	if width <= 0 {
		return line
	}

	return ansi.Truncate(line, width, "…")
}

// with returns a copy of the transfer test with a live update applied.
//
// Parameters:
//   - update: the live rate and latency.
//
// Returns:
//   - *transferView: the updated test, or nil when no test is running.
func (t *transferView) with(update updateTransfer) *transferView {
	if t == nil {
		return nil
	}

	updated := *t
	updated.rate, updated.latency, updated.remaining, updated.measured = update.rate, update.latency, update.remaining, true

	if total := update.elapsed + update.remaining; total > 0 {
		updated.done = max(t.done, min(float64(update.elapsed)/float64(total), 1))
	}

	return &updated
}

// timeLeft describes the estimated time left in a transfer test, in whole seconds rounded up.
//
// Parameters:
//   - remaining: the estimated time left.
//
// Returns:
//   - string: such as "~4s left", or "finishing" once nothing is left.
func timeLeft(remaining time.Duration) string {
	if remaining <= 0 {
		return "finishing"
	}

	return fmt.Sprintf("~%ds left", int(math.Ceil(remaining.Seconds())))
}

// progress returns the running test as a progress event, for its progress line.
//
// Returns:
//   - runner.TransferProgress: the direction, rate, and latency.
func (t *transferView) progress() runner.TransferProgress {
	return runner.TransferProgress{Direction: t.direction, Rate: t.rate, Latency: t.latency}
}

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
)

// model is the bubbletea model of the [TUI] reporter. It draws a spinner row for each running lookup or test.
// Finished lines are printed above it, so they stay in the terminal's scrollback.
type model struct {
	// rows are the running lookups and tests, in the order they started.
	rows []row
	// spinner animates the rows.
	spinner spinner.Model
	// width is the terminal width, or 0 while it is unknown.
	width int
	// quitting is set when the run finished, so the last frame leaves no rows behind.
	quitting bool
}

// row is a running lookup or test.
type row struct {
	// text describes it.
	text string
	// key identifies it.
	key rowKey
}

// rowKey identifies a row, so its updates and its end find it.
type rowKey int

// Row keys. The lookups of the caller and the servers run at the same time, and then one test runs at a time.
const (
	// userRow is the lookup of the caller.
	userRow rowKey = iota
	// serversRow is the lookup of the servers.
	serversRow
	// testRow is the running test.
	testRow
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
	// endRun ends the program, printing the rows that never finished as failed when the run failed.
	endRun struct {
		// failed is set when the run failed.
		failed bool
	}
)

// Styles of the marks and rows.
var (
	// doneMark marks a finished line.
	doneMark = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
	// failedMark marks a lookup or test that a failed run left unfinished.
	failedMark = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("✗")
	// spinnerStyle colors the spinner of a running row.
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

// newModel creates the model with no rows.
//
// Returns:
//   - model: the model.
func newModel() model {
	return model{
		rows: nil,
		spinner: spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(spinnerStyle),
		),
		width:    0,
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

// Update applies a message: a row change, the end of the run, a terminal resize, or a spinner tick.
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
	case endRun:
		return m.endRun(msg.failed)
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	default:
		// The program's other messages change nothing the rows show.
	}

	return m, nil
}

// View draws one line per running row: the spinner and the row's text, cut to the terminal width.
//
// Returns:
//   - tea.View: the rows, or nothing once the run finished.
func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	lines := make([]string, 0, len(m.rows))

	for _, running := range m.rows {
		lines = append(lines, m.fit(m.spinner.View()+" "+running.text))
	}

	return tea.NewView(strings.Join(lines, "\n"))
}

// endRun quits the program. When the run failed, the rows that never finished are printed as failed first.
//
// Parameters:
//   - failed: whether the run failed.
//
// Returns:
//   - tea.Model: the model, which draws nothing more.
//   - tea.Cmd: the prints, in order, and then quit.
func (m model) endRun(failed bool) (tea.Model, tea.Cmd) {
	m.quitting = true

	cmds := make([]tea.Cmd, 0, len(m.rows)+1)

	if failed {
		for _, unfinished := range m.rows {
			cmds = append(cmds, tea.Println(failedMark+" "+unfinished.text))
		}
	}

	m.rows = nil

	return m, tea.Sequence(append(cmds, tea.Quit)...)
}

// fit cuts a line to the terminal width, so a long row never wraps and breaks the redraw.
//
// Parameters:
//   - line: the styled line.
//
// Returns:
//   - string: the line, cut with an ellipsis when it is wider than the terminal.
func (m model) fit(line string) string {
	if m.width <= 0 {
		return line
	}

	return ansi.Truncate(line, m.width, "…")
}

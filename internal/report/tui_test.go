// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// update applies messages to a model in turn.
func update(t *testing.T, current model, msgs ...tea.Msg) model {
	t.Helper()

	for _, msg := range msgs {
		next, _ := current.Update(msg)

		updated, ok := next.(model)
		require.True(t, ok)

		current = updated
	}

	return current
}

// rowTexts returns the text of each drawn row, without its spinner.
func rowTexts(m model) []string {
	content := ansi.Strip(m.View().Content)
	if content == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	for index, line := range lines {
		_, text, _ := strings.Cut(line, " ")
		lines[index] = text
	}

	return lines
}

// TestModel_Rows checks that rows appear in the order they started, change with their updates, and leave when
// they end, and that the end of the run leaves nothing drawn.
func TestModel_Rows(t *testing.T) {
	t.Parallel()

	current := update(t, newModel(),
		startRow{text: userTitle, key: userRow},
		startRow{text: serversTitle, key: serversRow},
	)
	assert.Equal(t, []string{userTitle, serversTitle}, rowTexts(current))

	current = update(t, current, endRow{key: userRow}, startRow{text: pingTitle, key: testRow})
	assert.Equal(t, []string{serversTitle, pingTitle}, rowTexts(current))

	current = update(
		t,
		current,
		updateRow{text: "Latency: 4ms", key: testRow},
		endRow{key: serversRow},
	)
	assert.Equal(t, []string{"Latency: 4ms"}, rowTexts(current))

	current = update(t, current, endRun{failed: false})
	assert.Empty(t, rowTexts(current))
}

// TestModel_Width checks that a row wider than the terminal is cut to its width, so it never wraps.
func TestModel_Width(t *testing.T) {
	t.Parallel()

	current := update(t, newModel(),
		tea.WindowSizeMsg{Width: 20, Height: 10},
		startRow{text: "Download: 115.52 Mbps (Latency: 22ms)", key: testRow},
	)

	line := current.View().Content
	assert.Equal(t, 20, ansi.StringWidth(line))
	assert.True(t, strings.HasSuffix(ansi.Strip(line), "…"))
}

// runTUI reports events to a terminal reporter that draws into a buffer, and returns what it drew, without styles.
func runTUI(t *testing.T, events []runner.Event) string {
	t.Helper()

	var out bytes.Buffer

	reporter := NewTUI(
		&out,
		Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false},
		tea.WithWindowSize(120, 24),
	)

	report(reporter, events)

	return out.String()
}

// TestTUI checks a whole run: every finished line is printed, in the order of the events, with its mark, and the
// alternate screen is never used, so the results stay in the scrollback.
func TestTUI(t *testing.T) {
	t.Parallel()

	drawn := runTUI(t, runEvents())

	assert.NotContains(
		t,
		drawn,
		ansi.SetModeAltScreenSaveCursor,
		"the alternate screen must not be used",
	)

	plain := ansi.Strip(drawn)
	position := 0

	for _, line := range resultLines {
		if line == "" {
			continue
		}

		index := strings.Index(plain[position:], "✓ "+line)
		require.GreaterOrEqual(t, index, 0, "missing or out of order: %q", line)

		position += index + len(line)
	}
}

// TestTUI_Failed checks that a failed run prints the lookups or tests it left unfinished as failed.
func TestTUI_Failed(t *testing.T) {
	t.Parallel()

	drawn := ansi.Strip(runTUI(t, []runner.Event{
		runner.LookupStarted{},
		runner.UserFound{User: testUser},
		runner.RunFinished{Err: errors.New("connection refused")},
	}))

	assert.Contains(t, drawn, "✓ "+resultLines[0])
	assert.Contains(t, drawn, "✗ "+serversTitle)
	assert.NotContains(t, drawn, "✗ "+userTitle, "a finished lookup is not failed")
}

// TestTUI_ReportAfterEnd checks that events after the run finished are dropped instead of blocking.
func TestTUI_ReportAfterEnd(t *testing.T) {
	t.Parallel()

	reporter := NewTUI(
		&bytes.Buffer{},
		Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false},
	)
	reporter.Report(runner.RunFinished{Err: nil})

	assert.NotPanics(t, func() { reporter.Report(runner.PacketLossStarted{}) })
}

// TestTUI_Err checks that the reporter keeps the error that ended its drawing, so a run cannot look complete after
// its results were lost, and that a clean run has none.
func TestTUI_Err(t *testing.T) {
	t.Parallel()

	options := Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false}

	clean := NewTUI(&bytes.Buffer{}, options, tea.WithWindowSize(80, 24))
	require.NoError(t, clean.Err(), "a running program has no error")

	report(clean, runEvents())
	require.NoError(t, clean.Err())

	// A program whose context has ended stops at once, as one that fails to start does, and drops every event.
	ended, cancel := context.WithCancel(t.Context())
	cancel()

	failed := NewTUI(&bytes.Buffer{}, options, tea.WithWindowSize(80, 24), tea.WithContext(ended))
	report(failed, runEvents())

	err := failed.Err()
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to draw the progress")
}

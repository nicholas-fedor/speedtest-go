// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// testOptions format the lines in the tests.
var testOptions = Options{Unit: speedtest.UnitTypeDefaultMbps, MaskISP: false}

// testModel returns a model with nothing running.
func testModel() model {
	return newModel(testOptions, &atomic.Int64{})
}

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

	current := update(t, testModel(),
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

	current := update(t, testModel(),
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

// transferModel returns a model of an 80-column terminal running a download test, with the given updates applied.
func transferModel(t *testing.T, msgs ...tea.Msg) model {
	t.Helper()

	return update(t, testModel(), append([]tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: 24},
		startTransfer{direction: runner.Download},
	}, msgs...)...)
}

// progressAt returns a live update of a download at 100 Mbps with the given elapsed and remaining times.
func progressAt(elapsed, remaining time.Duration) updateTransfer {
	return updateTransfer{
		rate:      12_500_000,
		latency:   21 * time.Millisecond,
		elapsed:   elapsed,
		remaining: remaining,
	}
}

// TestModel_Transfer checks the row of a running transfer test: the direction alone until the first estimate, then
// the progress line, a bar of the share done, and the estimated time left, all on one line.
func TestModel_Transfer(t *testing.T) {
	t.Parallel()

	current := transferModel(t)
	assert.Equal(t, []string{"Download"}, rowTexts(current))

	current = update(t, current, progressAt(4*time.Second, 6*time.Second))
	assert.Equal(
		t,
		[]string{"Download: 100.00 Mbps (Latency: 21ms)  ━━━━━━━━────────────  ~6s left"},
		rowTexts(current),
	)

	current = update(t, current, endTransfer{})
	assert.Empty(t, current.View().Content)
}

// TestModel_TransferBar checks the bar: it fills with the share done of the estimated length of the test, it never
// moves back when the estimate grows, and it is full once nothing is left.
func TestModel_TransferBar(t *testing.T) {
	t.Parallel()

	filled := func(current model) int { return strings.Count(rowTexts(current)[0], "━") }

	current := transferModel(t, progressAt(6*time.Second, 4*time.Second))
	assert.Equal(t, 12, filled(current))

	current = update(t, current, progressAt(7*time.Second, 8*time.Second))
	assert.Equal(t, 12, filled(current), "a longer estimate must not move the bar back")
	assert.Contains(t, rowTexts(current)[0], "~8s left")

	current = update(t, current, progressAt(10*time.Second, 0))
	assert.Equal(t, barWidth, filled(current))
	assert.Contains(t, rowTexts(current)[0], "finishing")
}

// TestModel_TransferWithoutBar checks that a row leaves the bar out, rather than cutting it, when the terminal is too
// narrow for it.
func TestModel_TransferWithoutBar(t *testing.T) {
	t.Parallel()

	narrow := update(
		t,
		testModel(),
		tea.WindowSizeMsg{Width: 60, Height: 24},
		startTransfer{direction: runner.Upload},
		updateTransfer{
			rate:      500_000,
			latency:   21 * time.Millisecond,
			elapsed:   time.Second,
			remaining: 9 * time.Second,
		},
	)
	assert.Equal(t, []string{"Upload: 4.00 Mbps (Latency: 21ms)"}, rowTexts(narrow))
}

// TestTimeLeft checks that the time left is shown in whole seconds, rounded up so it never reads zero too early.
func TestTimeLeft(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "~6s left", timeLeft(6*time.Second))
	assert.Equal(t, "~6s left", timeLeft(5100*time.Millisecond))
	assert.Equal(t, "~1s left", timeLeft(time.Millisecond))
	assert.Equal(t, "finishing", timeLeft(0))
}

// TestSummaryPanel checks the panel of a server: the server in the frame, the headline results in two columns, N/A
// for what was not measured, and lines of equal width.
func TestSummaryPanel(t *testing.T) {
	t.Parallel()

	ping := speedtest.PingStats{Latency: 12207 * time.Microsecond}
	download := speedtest.Transfer{Rate: 86_676_250}
	loss := speedtest.PacketLossStats{Sent: 217, Dup: 0, Max: 237}

	panel := ansi.Strip(summaryPanel(speedtest.Result{
		Server:     testServer,
		Ping:       &ping,
		Download:   &download,
		Upload:     nil,
		PacketLoss: &loss,
	}, testOptions))

	lines := strings.Split(panel, "\n")
	require.Len(t, lines, 4)

	assert.True(t, strings.HasPrefix(lines[0], "╭─ sudosan · Shizuoka ─"))
	assert.Contains(t, lines[1], "Download    693.41 Mbps     Upload      N/A")
	assert.Contains(t, lines[2], "Latency     12.21 ms        Packet loss 8.82%")

	for _, line := range lines {
		assert.Equal(t, ansi.StringWidth(lines[0]), ansi.StringWidth(line), line)
	}
}

// TestTitledBorder checks that a title too long for the frame is cut instead of widening it.
func TestTitledBorder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "╭─ A long… ─╮", ansi.Strip(titledBorder("╭───────────╮", "A long title")))
	assert.Equal(t, "╭──╮", titledBorder("╭──╮", "Title"))
}

// TestTUI_Panel checks that a run ends each server with its panel in a wide terminal, and leaves it out in a narrow
// one.
func TestTUI_Panel(t *testing.T) {
	t.Parallel()

	wide := ansi.Strip(runTUI(t, runEvents()))
	assert.Contains(t, wide, "╭─ sudosan · Shizuoka")

	var out bytes.Buffer

	report(NewTUI(&out, testOptions, tea.WithWindowSize(40, 24)), runEvents())
	assert.NotContains(t, ansi.Strip(out.String()), "╭─")
}

// TestTUI_FailedTransfer checks that an interrupted transfer test is printed as failed, in the form of its progress
// line.
func TestTUI_FailedTransfer(t *testing.T) {
	t.Parallel()

	drawn := ansi.Strip(runTUI(t, []runner.Event{
		runner.TransferStarted{Direction: runner.Download},
		runner.TransferProgress{
			Direction: runner.Download,
			Rate:      12_500_000,
			Latency:   21 * time.Millisecond,
		},
		runner.RunFinished{Err: context.Canceled},
	}))

	assert.Contains(t, drawn, "✗ Download: 100.00 Mbps (Latency: 21ms)")
}

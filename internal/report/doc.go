// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package report shows the progress of a speed test run.
//
// Each reporter implements [runner.Reporter]: [TUI] shows live progress in a terminal, [Lines] writes one line per
// result for --unix output and for output that is not a terminal, and [Discard] shows nothing, for the JSON reports.
// The text of each line is the same in every reporter that shows it.
package report

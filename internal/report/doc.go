// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package report shows the progress of a speed test run.
//
// Each reporter implements [runner.Reporter]: [Spinner] shows live progress with spinners, [Lines] writes one line
// per result for --unix output, and [Discard] shows nothing, for the JSON reports. The text of each line is the same
// in every reporter that shows it.
package report

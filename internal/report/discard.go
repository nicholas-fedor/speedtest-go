// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package report

import (
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
)

// Discard shows nothing, so the JSON reports are the only output of a run.
type Discard struct{}

// Compile-time check that [Discard] is a [runner.Reporter].
var _ runner.Reporter = Discard{}

// Report ignores the event.
func (Discard) Report(runner.Event) {}

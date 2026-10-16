// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package runner runs the CLI's speed tests: it selects the servers, measures each one, and returns the results.
//
// A [Runner] never prints or exits. It describes its progress to a [Reporter] as [Event] values and returns its
// errors, so the CLI decides how a run looks and how it ends. It reaches the speedtest library through the small
// [Tester] interface, which [speedtest.Client] satisfies.
package runner

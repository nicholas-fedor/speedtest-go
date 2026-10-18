// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Command docgen writes the speedtest-go CLI reference as Hugo pages.
//
// It reads the command tree from [cmd.DocRoot], which builds the same tree the
// binary runs without any process dependencies. No command is executed, so
// docgen never contacts speedtest.net or reads the user's settings, and the
// reference cannot drift from the shipped command line.
//
// Commands and flags are visited in sorted order, so the same tree always
// produces the same bytes.
//
// Usage:
//
//	go run ./tools/docgen -out ./docs/content/usage/cli-reference
package main

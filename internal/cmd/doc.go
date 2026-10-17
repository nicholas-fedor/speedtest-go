// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package cmd builds the speedtest-go command tree.
//
// The commands parse their flags, resolve their settings, and pass them to the services they are given. They do not
// run tests, set up clients, or exit the process.
package cmd

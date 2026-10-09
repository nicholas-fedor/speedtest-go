// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package testserver provides a fake speedtest.net API and speedtest server for hermetic tests.
//
// One [API] serves both the speedtest.net endpoints that list servers and report the caller's details, and the
// per-server endpoints used for latency, download, and upload. Every listed server points back at the same fake,
// so a test can fetch servers and then measure against them without leaving the loopback interface.
//
// The package deliberately does not import the speedtest package, so the speedtest package's own tests can use it.
// It writes the JSON and XML wire formats itself.
package testserver

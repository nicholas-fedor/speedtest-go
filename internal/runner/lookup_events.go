// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// LookupStarted reports that the lookups of the caller and the servers started.
type LookupStarted struct{}

// UserFound reports the caller's details.
type UserFound struct {
	// User is the caller's details.
	User *speedtest.User
}

// ServersFound reports the servers the run tests against.
type ServersFound struct {
	// Codes are the country codes the listed servers were filtered by, with [CountryAuto] resolved, or nil.
	Codes []string
	// Count is the number of servers in the pool.
	Count int
	// Source is where the servers came from.
	Source ServerSource
}

// ServerSource is where the servers of a run came from.
type ServerSource int

// Sources of the servers of a run.
const (
	// SourceListed is the server list from speedtest.net.
	SourceListed ServerSource = iota
	// SourceSpecified is the servers named by ID.
	SourceSpecified
	// SourceCustom is a server named by URL.
	SourceCustom
)

// event marks [LookupStarted] as an event.
func (LookupStarted) event() {}

// event marks [UserFound] as an event.
func (UserFound) event() {}

// event marks [ServersFound] as an event.
func (ServersFound) event() {}

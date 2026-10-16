// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"sync"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Tester is the part of the speedtest client that a run uses.
type Tester interface {
	// FetchUserInfo returns the caller's details from speedtest.net.
	FetchUserInfo(ctx context.Context) (*speedtest.User, error)
	// FetchServers returns the listed servers with their probe latencies.
	FetchServers(ctx context.Context) (speedtest.Servers, error)
	// FetchServerByID returns the server with an ID.
	FetchServerByID(ctx context.Context, serverID string) (*speedtest.Server, error)
	// CustomServer returns a server for a URL that speedtest.net does not list.
	CustomServer(host string) (*speedtest.Server, error)
	// Ping measures the latency to a server.
	Ping(
		ctx context.Context,
		server *speedtest.Server,
		opts ...speedtest.PingOption,
	) (speedtest.PingStats, error)
	// Download measures the download speed from a server.
	Download(
		ctx context.Context,
		server *speedtest.Server,
		opts ...speedtest.TransferOption,
	) (speedtest.Transfer, error)
	// Upload measures the upload speed to a server.
	Upload(
		ctx context.Context,
		server *speedtest.Server,
		opts ...speedtest.TransferOption,
	) (speedtest.Transfer, error)
	// DownloadMulti measures the download speed from a main server and a pool of others at once.
	DownloadMulti(
		ctx context.Context,
		main *speedtest.Server,
		pool speedtest.Servers,
		opts ...speedtest.TransferOption,
	) (speedtest.Transfer, error)
	// UploadMulti measures the upload speed to a main server and a pool of others at once.
	UploadMulti(
		ctx context.Context,
		main *speedtest.Server,
		pool speedtest.Servers,
		opts ...speedtest.TransferOption,
	) (speedtest.Transfer, error)
	// PacketLoss measures the packet loss to a server until ctx ends.
	PacketLoss(ctx context.Context, server *speedtest.Server) (speedtest.PacketLossStats, error)
}

// Reporter receives the progress of a run. A run never calls it from two goroutines at once.
type Reporter interface {
	// Report receives one event.
	//
	// Parameters:
	//   - event: what happened.
	Report(event Event)
}

// Settings choose the servers and the tests of a run.
type Settings struct {
	// CustomURL is a server URL to test instead of a listed server, or empty.
	CustomURL string
	// City is the predefined city label whose country [CountryAuto] stands for, or empty for the user's own country.
	City string
	// ServerIDs are the listed servers to test, or empty to test the one with the lowest latency.
	ServerIDs []int
	// CountryCodes are the upper-case two-letter country codes the listed servers must be in, which may include
	// [CountryAuto]. Empty keeps every server.
	CountryCodes []string
	// Multi spreads each transfer test across the main server and the rest of the pool.
	Multi bool
	// NoDownload skips the download test.
	NoDownload bool
	// NoUpload skips the upload test.
	NoUpload bool
}

// Runner runs speed tests and reports their progress.
type Runner struct {
	// tester measures the servers.
	tester Tester
	// reporter receives the progress.
	reporter Reporter
	// settings choose the servers and the tests.
	settings Settings
	// mu keeps reports from overlapping, since callbacks and lookups run on their own goroutines.
	mu sync.Mutex
}

// CountryAuto is the country code that stands for the user's own country, or the country of [Settings.City].
const CountryAuto = "AUTO"

// Compile-time check that the speedtest client is a [Tester].
var _ Tester = (*speedtest.Client)(nil)

// New creates a runner.
//
// Parameters:
//   - tester: measures the servers.
//   - reporter: receives the progress.
//   - settings: choose the servers and the tests.
//
// Returns:
//   - *Runner: the runner.
func New(tester Tester, reporter Reporter, settings Settings) *Runner {
	return &Runner{tester: tester, reporter: reporter, settings: settings, mu: sync.Mutex{}}
}

// Run looks up the caller and the servers, then tests each target server in turn: latency, download, upload, and
// packet loss. The first failure ends the run. The last event it reports is always [RunFinished].
//
// Parameters:
//   - ctx: ends the run. A run it ends returns its error, so an interrupted run never looks complete.
//
// Returns:
//   - *speedtest.User: the caller's details.
//   - []speedtest.Result: the results of the servers tested so far.
//   - error: the error that ended the run, or nil when every test finished.
func (r *Runner) Run(ctx context.Context) (*speedtest.User, []speedtest.Result, error) {
	user, results, err := r.run(ctx)

	r.report(RunFinished{Err: err})

	return user, results, err
}

// run performs the run that [Runner.Run] describes, without its final event.
//
// Parameters:
//   - ctx: ends the run.
//
// Returns:
//   - *speedtest.User: the caller's details, or nil when the lookups failed.
//   - []speedtest.Result: the results of the servers tested so far.
//   - error: the error that ended the run.
func (r *Runner) run(ctx context.Context) (*speedtest.User, []speedtest.Result, error) {
	found, err := r.lookup(ctx)
	if err != nil {
		return nil, nil, err
	}

	results := make([]speedtest.Result, 0, len(found.targets))

	for _, server := range found.targets {
		result, err := r.testServer(ctx, server, found.pool)
		if err != nil {
			return found.user, results, err
		}

		results = append(results, result)
	}

	return found.user, results, nil
}

// report passes an event to the reporter, one at a time.
//
// Parameters:
//   - event: the event.
func (r *Runner) report(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reporter.Report(event)
}

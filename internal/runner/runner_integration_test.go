// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner/mocks"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// testServer is the one server the mocked lookups return.
var testServer = &speedtest.Server{ID: "1001", Latency: 1}

// expectLookups makes the mocked lookups find the caller and testServer.
func expectLookups(tester *mocks.MockTester) {
	tester.EXPECT().FetchUserInfo(mock.Anything).Return(&speedtest.User{IP: "203.0.113.7"}, nil)
	tester.EXPECT().FetchServers(mock.Anything).Return(speedtest.Servers{testServer}, nil)
}

// lastEvent returns the event the reporter mock received last.
func lastEvent(t *testing.T, reporter *mocks.MockReporter) runner.Event {
	t.Helper()

	calls := reporter.Calls
	require.NotEmpty(t, calls)

	event, ok := calls[len(calls)-1].Arguments.Get(0).(runner.Event)
	require.True(t, ok)

	return event
}

// TestRun_UserLookupFails checks that a failed user lookup ends the run before any server is tested, with an error
// that names the lookup, and that RunFinished carries it.
func TestRun_UserLookupFails(t *testing.T) {
	t.Parallel()

	failure := errors.New("connection refused")

	tester := mocks.NewMockTester(t)
	tester.EXPECT().FetchUserInfo(mock.Anything).Return(nil, failure)
	tester.EXPECT().FetchServers(mock.Anything).Return(speedtest.Servers{testServer}, nil)

	reporter := mocks.NewMockReporter(t)
	reporter.EXPECT().Report(mock.Anything)

	_, results, err := runner.New(tester, reporter, runner.Settings{}).Run(t.Context())
	require.ErrorIs(t, err, failure)
	assert.Contains(t, err.Error(), "failed to retrieve user information")
	assert.Empty(t, results)

	reporter.AssertNotCalled(t, "Report", runner.ServerStarted{Server: testServer})
	assert.Equal(t, runner.RunFinished{Err: err}, lastEvent(t, reporter))
}

// TestRun_PingFails checks that a failed latency measurement ends the run before the transfer and packet loss
// measurements start.
func TestRun_PingFails(t *testing.T) {
	t.Parallel()

	failure := errors.New("no route to host")

	tester := mocks.NewMockTester(t)
	expectLookups(tester)
	tester.EXPECT().
		Ping(mock.Anything, testServer, mock.Anything).
		Return(speedtest.PingStats{}, failure)

	reporter := mocks.NewMockReporter(t)
	reporter.EXPECT().Report(mock.Anything)

	_, _, err := runner.New(tester, reporter, runner.Settings{}).Run(t.Context())
	require.ErrorIs(t, err, failure)
	assert.Contains(t, err.Error(), "failed to measure the latency to server 1001")

	reporter.AssertNotCalled(t, "Report", runner.PacketLossStarted{})
}

// TestRun_InterruptedDuringPacketLoss checks that a run interrupted while only packet loss is measured returns the
// context's error and reports no packet loss, since the measurement ends quietly with the context.
func TestRun_InterruptedDuringPacketLoss(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tester := mocks.NewMockTester(t)
	expectLookups(tester)
	tester.EXPECT().
		Ping(mock.Anything, testServer, mock.Anything).
		Return(speedtest.PingStats{}, nil)
	tester.EXPECT().PacketLoss(mock.Anything, testServer).RunAndReturn(
		func(lossCtx context.Context, _ *speedtest.Server) (speedtest.PacketLossStats, error) {
			cancel()
			<-lossCtx.Done()

			return speedtest.PacketLossStats{Sent: 10, Dup: 0, Max: 9}, nil
		},
	)

	reporter := mocks.NewMockReporter(t)
	reporter.EXPECT().Report(mock.Anything)

	_, results, err := runner.New(
		tester,
		reporter,
		runner.Settings{NoDownload: true, NoUpload: true},
	).Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, results)

	reporter.AssertNotCalled(t, "Report", mock.AnythingOfType("runner.PacketLossFinished"))
}

// TestRun_Multi checks that a multi-server run passes the whole pool to the multi-server transfer tests.
func TestRun_Multi(t *testing.T) {
	t.Parallel()

	tester := mocks.NewMockTester(t)
	expectLookups(tester)
	tester.EXPECT().
		Ping(mock.Anything, testServer, mock.Anything).
		Return(speedtest.PingStats{}, nil).
		Maybe()
	tester.EXPECT().PacketLoss(mock.Anything, testServer).Return(speedtest.PacketLossStats{}, nil)
	tester.EXPECT().
		DownloadMulti(mock.Anything, testServer, speedtest.Servers{testServer}, mock.Anything).
		Return(speedtest.Transfer{}, nil)
	tester.EXPECT().
		UploadMulti(mock.Anything, testServer, speedtest.Servers{testServer}, mock.Anything).
		Return(speedtest.Transfer{}, nil)

	reporter := mocks.NewMockReporter(t)
	reporter.EXPECT().Report(mock.Anything)

	_, results, err := runner.New(tester, reporter, runner.Settings{Multi: true}).Run(t.Context())
	require.NoError(t, err)
	require.Len(t, results, 1)
}

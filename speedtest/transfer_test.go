// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/internal/meter"
)

// TestNew_MeterSettings checks that each test's meter gets the client's connection count, test duration, and logger:
// a download against the fake server runs two workers on its one server, logs them, and stops after the duration.
func TestNew_MeterSettings(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	logger := &recordingLogger{}
	client := newTestClient(
		t,
		WithConnections(2),
		WithTestDuration(200*time.Millisecond),
		WithLogger(logger),
	)

	transfer, err := client.Download(
		context.Background(),
		&Server{ID: "1001", URL: api.ServerUploadURL("1001")},
	)
	require.NoError(t, err)

	assert.Less(t, transfer.Duration, 5*time.Second, "the test stops after its duration")
	assert.Equal(t, 200*time.Millisecond, transfer.Window.Round(100*time.Millisecond))
	assert.Len(t, transfer.Connections, 2)

	logger.mu.Lock()
	defer logger.mu.Unlock()

	assert.Equal(
		t,
		[]any{"handlers", 1, "mainWorkers", 2, "otherWorkers", 0},
		logger.args["starting transfer workers"],
	)
}

// TestClient_Transfers checks that a download and an upload against the fake server report a rate, bytes that add up
// over their connections, live rates to their own callback, and an upload confirmation ratio, without changing the
// server.
func TestClient_Transfers(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t, WithConnections(2), WithTestDuration(300*time.Millisecond))
	server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Latency: time.Millisecond}
	before := *server

	for name, run := range map[string]func(...TransferOption) (Transfer, error){
		"download": func(opts ...TransferOption) (Transfer, error) {
			return client.Download(context.Background(), server, opts...)
		},
		"upload": func(opts ...TransferOption) (Transfer, error) {
			return client.Upload(context.Background(), server, opts...)
		},
	} {
		var (
			mu    sync.Mutex
			rates []ByteRate
		)

		transfer, err := run(WithProgressCallback(func(progress TransferProgress) {
			mu.Lock()
			defer mu.Unlock()

			rates = append(rates, progress.Rate)
		}))
		require.NoError(t, err, name)

		assert.Positive(t, float64(transfer.Rate), name)
		assert.Positive(t, transfer.Bytes, name)
		assertConnectionsAddUp(t, transfer)

		mu.Lock()
		assert.NotEmpty(t, rates, "%s: the callback receives live rates", name)
		mu.Unlock()

		if name == "upload" {
			assert.GreaterOrEqual(t, transfer.Sent, transfer.Bytes)
			assert.Positive(t, transfer.ConfirmationRatio())
		} else {
			assert.Zero(t, transfer.Sent)
			assert.Zero(t, transfer.ConfirmationRatio())
		}
	}

	assert.Equal(t, before, *server, "the tests must not change the server")
}

// TestClient_TransfersAreIndependent checks that each test has its own meter: of two concurrent downloads on one
// client, the one whose server answers only with errors counts nothing while the other counts data, each callback
// hears only its own test, and a second download in a row starts from zero.
func TestClient_TransfersAreIndependent(t *testing.T) {
	t.Parallel()

	failing := testserver.NewAPI(t)
	failing.SetResponse(
		testserver.PathDownload,
		testserver.Response{Status: http.StatusInternalServerError},
	)

	working := testserver.NewAPI(t)
	client := newTestClient(t, WithConnections(1), WithTestDuration(300*time.Millisecond))

	type outcome struct {
		err      error
		transfer Transfer
		rates    []ByteRate
	}

	run := func(api *testserver.API) outcome {
		var (
			mu    sync.Mutex
			rates []ByteRate
		)

		transfer, err := client.Download(
			context.Background(),
			&Server{ID: "1001", URL: api.ServerUploadURL("1001")},
			WithProgressCallback(func(progress TransferProgress) {
				mu.Lock()
				defer mu.Unlock()

				rates = append(rates, progress.Rate)
			}),
		)

		mu.Lock()
		defer mu.Unlock()

		return outcome{err: err, transfer: transfer, rates: rates}
	}

	var (
		wg                       sync.WaitGroup
		fromFailing, fromWorking outcome
	)

	wg.Go(func() { fromFailing = run(failing) })
	wg.Go(func() { fromWorking = run(working) })
	wg.Wait()

	require.NoError(t, fromFailing.err)
	require.NoError(t, fromWorking.err)

	assert.Zero(
		t,
		fromFailing.transfer.Bytes,
		"the failing test must not count the other test's data",
	)
	assert.InDelta(t, -1, float64(fromFailing.transfer.Rate), 1e-9)
	assert.Positive(t, fromWorking.transfer.Bytes)

	for _, rate := range fromFailing.rates {
		assert.Zero(t, rate, "the failing test's callback must not hear the other test's rate")
	}

	assert.NotEmpty(t, fromWorking.rates)

	again := run(working)
	require.NoError(t, again.err)
	assert.Positive(t, again.transfer.Bytes)
	assertConnectionsAddUp(t, again.transfer)
}

// TestClient_MultiConnections checks that a multi-server test names each connection's server, with the main server
// first, and counts each connection's bytes toward the total.
func TestClient_MultiConnections(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t, WithConnections(4), WithTestDuration(300*time.Millisecond))
	pool := Servers{
		{ID: "1001", URL: api.ServerUploadURL("1001"), Latency: 2 * time.Millisecond},
		{ID: "1002", URL: api.ServerUploadURL("1002"), Latency: time.Millisecond},
	}

	transfer, err := client.DownloadMulti(context.Background(), pool[0], pool)
	require.NoError(t, err)

	servers := make([]string, 0, len(transfer.Connections))
	for _, connection := range transfer.Connections {
		servers = append(servers, connection.ServerID)

		assert.Positive(t, connection.Requests)
		assert.LessOrEqual(t, connection.Active, transfer.Window)
	}

	assert.Equal(t, []string{"1001", "1002", "1002", "1002"}, servers)
	assertConnectionsAddUp(t, transfer)
}

// TestClient_MultiRejectsUnavailablePool checks that a multi-server test needs a server with a measured latency.
func TestClient_MultiRejectsUnavailablePool(t *testing.T) {
	t.Parallel()

	server := &Server{
		ID:      "1001",
		URL:     "http://127.0.0.1/speedtest/upload.php",
		Latency: PingTimeout,
	}

	_, err := newTestClient(t).DownloadMulti(context.Background(), server, Servers{server})
	require.ErrorIs(t, err, ErrNoAvailableServers)
}

// TestTransfer_ConfirmationRatio checks the share of sent upload bytes that were confirmed, which stays within 0 and
// 1 and is 0 when nothing was sent.
func TestTransfer_ConfirmationRatio(t *testing.T) {
	t.Parallel()

	assert.Zero(t, Transfer{Bytes: 100}.ConfirmationRatio())
	assert.InDelta(t, 0.25, Transfer{Bytes: 25, Sent: 100}.ConfirmationRatio(), 1e-9)
	assert.InDelta(t, 1.0, Transfer{Bytes: 200, Sent: 100}.ConfirmationRatio(), 1e-9)
}

// TestConnectionStats_Rate checks a connection's rate over the time it spent in requests.
func TestConnectionStats_Rate(t *testing.T) {
	t.Parallel()

	assert.InDelta(
		t,
		500,
		float64(ConnectionStats{Bytes: 1000, Active: 2 * time.Second}.Rate()),
		1e-9,
	)
	assert.Zero(t, ConnectionStats{Bytes: 1000}.Rate())
}

// TestNewTransfer_NotAvailable checks that a test that counted nothing reports N/A when more than a tenth of its
// requests failed, and zero otherwise.
func TestNewTransfer_NotAvailable(t *testing.T) {
	t.Parallel()

	failing := newTransfer(meterResult(0, 10, 2), nil, time.Second)
	assert.InDelta(t, -1, float64(failing.Rate), 1e-9)

	quiet := newTransfer(meterResult(0, 10, 1), nil, time.Second)
	assert.Zero(t, quiet.Rate)
}

// assertConnectionsAddUp checks that a transfer's connections count exactly its bytes, requests, and failures.
//
// Parameters:
//   - t: the test.
//   - transfer: the transfer.
func assertConnectionsAddUp(t *testing.T, transfer Transfer) {
	t.Helper()

	var (
		bytes    int64
		requests int
		failures int
	)

	for _, connection := range transfer.Connections {
		bytes += connection.Bytes
		requests += connection.Requests
		failures += connection.Failures
	}

	assert.Equal(t, transfer.Bytes, bytes, "the connections' bytes add up to the transfer's")
	assert.Equal(t, transfer.Requests, requests)
	assert.Equal(t, transfer.Failures, failures)
}

// meterResult returns a meter measurement with no connections.
//
// Parameters:
//   - bytes: the bytes counted.
//   - requests: the requests sent.
//   - failures: the requests that failed.
//
// Returns:
//   - meter.Result: the measurement.
func meterResult(bytes int64, requests, failures int) meter.Result {
	return meter.Result{
		Connections: nil,
		Rate:        0,
		Bytes:       bytes,
		Sent:        0,
		Window:      time.Second,
		Requests:    requests,
		Failures:    failures,
	}
}

// TestClient_MeasurementsRejectMissingArguments checks that every measurement reports a nil server, and a client
// that was not built by New, instead of panicking.
func TestClient_MeasurementsRejectMissingArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := newTestClient(t)
	server := &Server{ID: "1", URL: "http://127.0.0.1/speedtest/upload.php"}

	measurements := map[string]func(*Client, *Server) error{
		"ping": func(c *Client, s *Server) error {
			_, err := c.Ping(ctx, s)

			return err
		},
		"download": func(c *Client, s *Server) error {
			_, err := c.Download(ctx, s)

			return err
		},
		"upload": func(c *Client, s *Server) error {
			_, err := c.Upload(ctx, s)

			return err
		},
		"multi-server download": func(c *Client, s *Server) error {
			_, err := c.DownloadMulti(ctx, s, Servers{server})

			return err
		},
		"multi-server upload": func(c *Client, s *Server) error {
			_, err := c.UploadMulti(ctx, s, Servers{server})

			return err
		},
		"packet loss": func(c *Client, s *Server) error {
			_, err := c.PacketLoss(ctx, s)

			return err
		},
	}

	for name, measure := range measurements {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, measure(client, nil), ErrServerNil)
			require.ErrorIs(t, measure(nil, server), ErrUninitializedManager)
			require.ErrorIs(t, measure(&Client{}, server), ErrUninitializedManager)
		})
	}
}

// TestClient_Download_ErrorStatus checks that a server answering downloads with an error status ends with a
// download speed of N/A instead of a rate measured from error pages.
func TestClient_Download_ErrorStatus(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	api.SetResponse(
		testserver.PathDownload,
		testserver.Response{Status: http.StatusInternalServerError, Body: "error page"},
	)

	client := newTestClient(t, WithConnections(2), WithTestDuration(300*time.Millisecond))

	server := &Server{URL: api.ServerUploadURL("1001")}

	transfer, err := client.Download(context.Background(), server)
	require.NoError(t, err)

	assert.InDelta(
		t,
		-1,
		float64(transfer.Rate),
		1e-9,
		"a test of only failed requests reports N/A",
	)
	assert.Zero(t, transfer.Bytes)
	assert.Equal(t, transfer.Requests, transfer.Failures)
}

// TestClient_Transfers_Cancel checks that cancelling a download or upload ends it well before the capture time with
// the context's error and no result, and leaves the server unchanged.
func TestClient_Transfers_Cancel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		run  func(*Client, context.Context, *Server) (Transfer, error)
		name string
	}{
		{
			name: "download",
			run: func(c *Client, ctx context.Context, s *Server) (Transfer, error) {
				return c.Download(ctx, s)
			},
		},
		{
			name: "upload",
			run: func(c *Client, ctx context.Context, s *Server) (Transfer, error) {
				return c.Upload(ctx, s)
			},
		},
		{
			name: "multi download",
			run: func(c *Client, ctx context.Context, s *Server) (Transfer, error) {
				return c.DownloadMulti(ctx, s, Servers{s})
			},
		},
		{
			name: "multi upload",
			run: func(c *Client, ctx context.Context, s *Server) (Transfer, error) {
				return c.UploadMulti(ctx, s, Servers{s})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			server := &Server{
				ID:      "1001",
				URL:     api.ServerUploadURL("1001"),
				Latency: time.Millisecond,
			}
			before := *server

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			start := time.Now()
			transfer, err := tt.run(newTestClient(t, WithConnections(2)), ctx, server)

			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Less(t, time.Since(start), 2*time.Second, "the default capture time is 15s")
			assert.Equal(t, Transfer{}, transfer, "a cancelled test reports no result")
			assert.Equal(t, before, *server, "a test must not change the server")
		})
	}
}

// TestClient_Upload_Redirect checks that a full upload test against a redirecting server measures a rate, without
// ever posting to the redirecting URL, and without changing the server.
func TestClient_Upload_Redirect(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	moved := newRedirector(t, api.ServerUploadURL("1001"), http.StatusTemporaryRedirect)

	client := newTestClient(t, WithConnections(2), WithTestDuration(300*time.Millisecond))
	server := &Server{URL: moved.URL()}

	transfer, err := client.Upload(context.Background(), server)
	require.NoError(t, err)

	assert.Positive(t, float64(transfer.Rate))
	assert.Positive(t, transfer.Bytes)
	assert.Equal(t, []string{http.MethodHead}, moved.Methods())
	assert.Equal(t, moved.URL(), server.URL)
}

// TestClient_UploadMulti_Redirect checks that multi-server uploads resolve every server in the pool before the
// workers start, so no upload posts to a redirecting URL.
func TestClient_UploadMulti_Redirect(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	movedA := newRedirector(t, api.ServerUploadURL("1001"), http.StatusTemporaryRedirect)
	movedB := newRedirector(t, api.ServerUploadURL("1002"), http.StatusFound)

	client := newTestClient(t, WithConnections(2), WithTestDuration(300*time.Millisecond))

	pool := Servers{
		{ID: "1001", URL: movedA.URL()},
		{ID: "1002", URL: movedB.URL()},
	}

	transfer, err := client.UploadMulti(context.Background(), pool[0], pool)
	require.NoError(t, err)

	assert.Positive(t, float64(transfer.Rate))
	assert.Equal(t, []string{http.MethodHead}, movedA.Methods())
	assert.Equal(t, []string{http.MethodHead}, movedB.Methods())
}

// TestClient_transferFinalRate checks that download and upload tests report the measured volume over the capture
// window as their rate. One megabyte arrives every 100 ms, ten times in a 1.05 s window.
func TestClient_transferFinalRate(t *testing.T) {
	t.Parallel()

	request := func(_ context.Context, _ *transferRequest, connection *meter.Connection) error {
		time.Sleep(100 * time.Millisecond)
		connection.Confirm(1_000_000)

		return nil
	}

	synctest.Test(t, func(t *testing.T) {
		client := newTestClient(t, WithConnections(1), WithTestDuration(1050*time.Millisecond))
		server := &Server{ID: "1001"}

		for _, kind := range []transferKind{client.downloadKind(), client.uploadKind()} {
			kind.request = request
			kind.resolveUploads = false

			transfer, err := client.transfer(context.Background(), kind, Servers{server}, 0, nil)
			require.NoError(t, err)

			assert.InDelta(t, 10_000_000/1.05, float64(transfer.Rate), 1e-6)
			assert.Equal(t, int64(10_000_000), transfer.Bytes)
			assert.Equal(t, 1050*time.Millisecond, transfer.Window)
		}
	})
}

// TestMultiServers checks that the main server comes first, so it gets a handler however few connections a test
// has, while the others keep their order, fastest first, and the pool is left as it was.
func TestMultiServers(t *testing.T) {
	t.Parallel()

	pool := Servers{
		{ID: "1", Latency: 5 * time.Millisecond},
		{ID: "2", Latency: time.Millisecond},
		{ID: "3", Latency: PingTimeout},
		{ID: "4", Latency: 3 * time.Millisecond},
		{ID: "5", Latency: 2 * time.Millisecond},
	}
	before := slices.Clone(pool)

	ids := func(servers Servers) []string {
		result := make([]string, 0, len(servers))
		for _, server := range servers {
			result = append(result, server.ID)
		}

		return result
	}

	servers, err := multiServers(pool[0], pool)
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "5", "4"}, ids(servers))
	assert.Equal(t, before, pool)

	servers, err = multiServers(&Server{ID: "9"}, pool)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"2", "5", "4", "1"},
		ids(servers),
		"a main server outside the pool changes nothing",
	)
}

// TestClient_MultiMainBeyondConnections checks that the slowest server of a pool gets the main connection when it is
// the main server, even when the test has fewer connections than the pool has servers.
func TestClient_MultiMainBeyondConnections(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	pool := make(Servers, 0, 4)

	for index := range 4 {
		id := strconv.Itoa(1001 + index)
		pool = append(
			pool,
			&Server{
				ID:      id,
				URL:     api.ServerUploadURL(id),
				Latency: time.Duration(index+1) * time.Millisecond,
			},
		)
	}

	client := newTestClient(t, WithConnections(2), WithTestDuration(200*time.Millisecond))

	transfer, err := client.DownloadMulti(context.Background(), pool[3], pool)
	require.NoError(t, err)

	require.NotEmpty(t, transfer.Connections)
	assert.Equal(t, "1004", transfer.Connections[0].ServerID)
}

// TestClient_UploadMulti_ResolvesOnlyUsedServers checks that a multi-server upload resolves the upload URLs of only
// the servers that get a connection, not the whole pool.
func TestClient_UploadMulti_ResolvesOnlyUsedServers(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	pool := make(Servers, 0, 4)
	redirectors := make([]*redirector, 0, 4)

	for index := range 4 {
		id := strconv.Itoa(1001 + index)
		moved := newRedirector(t, api.ServerUploadURL(id), http.StatusFound)
		redirectors = append(redirectors, moved)
		pool = append(
			pool,
			&Server{ID: id, URL: moved.URL(), Latency: time.Duration(index+1) * time.Millisecond},
		)
	}

	client := newTestClient(t, WithConnections(2), WithTestDuration(200*time.Millisecond))

	_, err := client.UploadMulti(context.Background(), pool[0], pool)
	require.NoError(t, err)

	assert.Equal(t, []string{http.MethodHead}, redirectors[0].Methods())
	assert.Equal(t, []string{http.MethodHead}, redirectors[1].Methods())
	assert.Empty(t, redirectors[2].Methods(), "a server without a connection must not be resolved")
	assert.Empty(t, redirectors[3].Methods(), "a server without a connection must not be resolved")
}

// Test_progressCallback checks that the meter's progress reaches the callback as a rate, the elapsed time, and the
// time left, and that no callback adapts to none.
func Test_progressCallback(t *testing.T) {
	t.Parallel()

	assert.Nil(t, progressCallback(nil))

	var got TransferProgress

	progressCallback(func(progress TransferProgress) { got = progress })(meter.Progress{
		Rate:      12_500_000,
		Elapsed:   4 * time.Second,
		Remaining: 6 * time.Second,
	})

	assert.Equal(
		t,
		TransferProgress{Rate: 12_500_000, Elapsed: 4 * time.Second, Remaining: 6 * time.Second},
		got,
	)
}

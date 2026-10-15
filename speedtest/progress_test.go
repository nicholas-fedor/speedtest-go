// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// TestNew_MeterSettings checks that New hands the connection count, test duration, and logger to the meter: a
// download against the fake server runs two workers on its one server, logs them, and stops after the duration.
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
	server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}

	start := time.Now()

	require.NoError(t, server.DownloadTestContext(context.Background()))
	assert.Less(t, time.Since(start), 5*time.Second, "the test stops after its duration")

	logger.mu.Lock()
	defer logger.mu.Unlock()

	assert.Equal(
		t,
		[]any{"handlers", 1, "mainWorkers", 2, "otherWorkers", 0},
		logger.args["starting transfer workers"],
	)
}

// TestClient_Progress checks that the rate callbacks reach the meter, that the totals and the confirmation ratio
// report its counts, and that Reset clears them.
func TestClient_Progress(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	client := newTestClient(t, WithConnections(1), WithTestDuration(300*time.Millisecond))
	server := &Server{ID: "1001", URL: api.ServerUploadURL("1001"), Context: client}

	rates := make(chan ByteRate, 1000)

	client.SetCallbackDownload(func(rate ByteRate) { rates <- rate })
	client.SetCallbackUpload(func(rate ByteRate) { rates <- rate })

	require.NoError(t, server.DownloadTestContext(context.Background()))
	require.NoError(t, server.UploadTestContext(context.Background()))

	assert.NotEmpty(t, rates, "the callbacks receive live rates")
	assert.Positive(t, client.GetTotalDownload())
	assert.Positive(t, client.GetTotalUpload())
	assert.Positive(t, client.GetUploadConfirmationRatio())

	client.Reset()

	assert.Zero(t, client.GetTotalDownload())
	assert.Zero(t, client.GetTotalUpload())
	assert.Zero(t, client.GetUploadConfirmationRatio())
}

// TestClient_ProgressZeroValue checks that the progress methods of a client built without New do nothing instead of
// panicking, and that its servers report that they cannot measure.
func TestClient_ProgressZeroValue(t *testing.T) {
	t.Parallel()

	for _, client := range []*Client{nil, {}} {
		assert.NotPanics(t, func() {
			client.SetCallbackDownload(func(ByteRate) {})
			client.SetCallbackUpload(nil)
			client.Reset()
		})

		assert.Zero(t, client.GetTotalDownload())
		assert.Zero(t, client.GetTotalUpload())
		assert.Zero(t, client.GetUploadConfirmationRatio())

		server := &Server{URL: "http://127.0.0.1/speedtest/upload.php", Context: client}

		require.ErrorIs(
			t,
			server.DownloadTestContext(context.Background()),
			ErrUninitializedManager,
		)
		require.ErrorIs(t, server.UploadTestContext(context.Background()), ErrUninitializedManager)
		require.ErrorIs(
			t,
			server.MultiDownloadTestContext(context.Background(), Servers{server}),
			ErrUninitializedManager,
		)
		require.ErrorIs(
			t,
			server.MultiUploadTestContext(context.Background(), Servers{server}),
			ErrUninitializedManager,
		)
		require.ErrorIs(
			t,
			downloadRequest(context.Background(), server, 0),
			ErrUninitializedManager,
		)
		require.ErrorIs(t, uploadRequest(context.Background(), server, 0), ErrUninitializedManager)
	}
}

package echo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// newClient returns a speedtest client with the default settings.
//
// Parameters:
//   - t: the test, which fails when the client cannot be built.
//
// Returns:
//   - *speedtest.Client: the client.
func newClient(t *testing.T) *speedtest.Client {
	t.Helper()

	client, err := speedtest.New()
	require.NoError(t, err)

	return client
}

// TestAccompanyEcho_RecordsLatency checks that the echo pings the server through the client while it runs and keeps
// each latency it measures.
func TestAccompanyEcho_RecordsLatency(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	server := &speedtest.Server{ID: "1001", URL: api.ServerUploadURL("1001")}
	echo := New(newClient(t), server, 20*time.Millisecond)

	echo.Run()
	defer echo.Stop()

	require.Eventually(
		t,
		func() bool { return len(echo.Latencies()) > 0 },
		5*time.Second,
		10*time.Millisecond,
	)
	assert.Positive(t, echo.CurrentLatency())
	assert.True(t, api.Requested(testserver.PathLatency))
}

func TestNew(t *testing.T) {
	t.Parallel()

	type args struct {
		server   *speedtest.Server
		interval time.Duration
	}

	tests := []struct {
		name string
		args args
		want *AccompanyEcho
	}{
		{
			name: "create accompany echo",
			args: args{
				server:   &speedtest.Server{ID: "1"},
				interval: time.Second,
			},
			want: &AccompanyEcho{
				client:   nil,
				server:   &speedtest.Server{ID: "1"},
				interval: time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t)
			got := New(client, tt.args.server, tt.args.interval)
			assert.NotNil(t, got)
			assert.Same(t, client, got.client)
			assert.Equal(t, tt.args.server, got.server)
			assert.Equal(t, tt.args.interval, got.interval)
		})
	}
}

func TestAccompanyEcho_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ae   *AccompanyEcho
	}{
		{
			name: "run accompany echo",
			ae:   New(nil, &speedtest.Server{ID: "1"}, time.Millisecond*100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.ae.Run() })
			time.Sleep(10 * time.Millisecond) // let it start
			assert.NotPanics(t, func() { tt.ae.Stop() })
		})
	}
}

func TestAccompanyEcho_Stop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ae   *AccompanyEcho
	}{
		{
			name: "stop accompany echo",
			ae:   New(nil, &speedtest.Server{ID: "1"}, time.Millisecond*100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.ae.Run() })
			time.Sleep(10 * time.Millisecond)
			assert.NotPanics(t, func() { tt.ae.Stop() })
		})
	}
}

func TestAccompanyEcho_CurrentLatency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ae   *AccompanyEcho
		want int64
	}{
		{
			name: "initial latency",
			ae:   New(nil, &speedtest.Server{ID: "1"}, time.Millisecond*100),
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.ae.CurrentLatency()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAccompanyEcho_Latencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ae   *AccompanyEcho
		want []int64
	}{
		{
			name: "initial latencies",
			ae:   New(nil, &speedtest.Server{ID: "1"}, time.Millisecond*100),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.ae.Latencies()
			assert.Equal(t, tt.want, got)
		})
	}
}

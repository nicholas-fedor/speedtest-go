package speedtest

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_checkStatus checks which statuses pass and what a failing status reports.
func Test_checkStatus(t *testing.T) {
	t.Parallel()

	requested, err := url.Parse("https://user:secret@speedtest.example/speedtest/latency.txt")
	require.NoError(t, err)

	tests := []struct {
		request *http.Request
		name    string
		wantURL string
		status  int
		wantErr bool
	}{
		{name: "200 OK", status: http.StatusOK},
		{name: "204 No Content", status: http.StatusNoContent},
		{name: "299 is still 2xx", status: 299},
		{
			name:    "unfollowed redirect",
			status:  http.StatusFound,
			request: &http.Request{URL: requested},
			wantErr: true,
			wantURL: "https://user:xxxxx@speedtest.example/speedtest/latency.txt",
		},
		{
			name:    "not found",
			status:  http.StatusNotFound,
			request: &http.Request{URL: requested},
			wantErr: true,
		},
		{
			name:    "server error without a request",
			status:  http.StatusInternalServerError,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkStatus(&http.Response{StatusCode: tt.status, Request: tt.request})
			if !tt.wantErr {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, ErrUnexpectedStatus)

			var statusErr *StatusError
			require.ErrorAs(t, err, &statusErr)
			assert.Equal(t, tt.status, statusErr.StatusCode)
			assert.NotContains(t, statusErr.URL, "secret", "the URL must be redacted")

			if tt.wantURL != "" {
				assert.Equal(t, tt.wantURL, statusErr.URL)
			}
		})
	}
}

// TestStatusError_Error checks the message format and that wrapping keeps the sentinel reachable.
func TestStatusError_Error(t *testing.T) {
	t.Parallel()

	err := &StatusError{
		URL:        "https://speedtest.example/api/js/servers",
		StatusCode: http.StatusServiceUnavailable,
	}

	assert.Equal(
		t,
		"unexpected HTTP status: 503 Service Unavailable from https://speedtest.example/api/js/servers",
		err.Error(),
	)
	assert.ErrorIs(t, err, ErrUnexpectedStatus)
}

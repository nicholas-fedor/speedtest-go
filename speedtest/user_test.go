// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

// TestClient_FetchUserInfo covers decoding the user info response and failures.
func TestClient_FetchUserInfo(t *testing.T) {
	t.Parallel()

	want := testserver.DefaultUser()

	tests := []struct {
		override  *testserver.Response
		want      *User
		wantErr   error
		name      string
		anyErr    bool
		cancelled bool
	}{
		{
			name: "reports and stores the user",
			want: &User{
				IP:      want.IP,
				Lat:     want.Lat,
				Lon:     want.Lon,
				Isp:     want.ISP,
				Country: want.Country,
			},
		},
		{
			name:     "response without a client element",
			override: &testserver.Response{Body: "<settings></settings>"},
			wantErr:  ErrFetchUserInfo,
		},
		{
			name:     "malformed XML",
			override: &testserver.Response{Body: "<settings><client"},
			anyErr:   true,
		},
		{
			name: "error status",
			override: &testserver.Response{
				Status: http.StatusServiceUnavailable,
				Body:   "unavailable",
			},
			wantErr: ErrUnexpectedStatus,
		},
		{
			name:      "cancelled context",
			cancelled: true,
			wantErr:   context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			if tt.override != nil {
				api.SetResponse(testserver.PathUserConfig, *tt.override)
			}

			client := newBaseURLClient(t, api.URL())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelled {
				cancel()
			}

			got, err := client.FetchUserInfo(ctx)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			case tt.anyErr:
				require.Error(t, err)
				assert.Nil(t, got)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		_, err := (*Client)(nil).FetchUserInfo(context.Background())
		require.ErrorIs(t, err, ErrInstanceNil)
	})
}

// TestUser_String checks the one-line user summary printed by the CLI.
func TestUser_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		u    *User
		want string
	}{
		{
			name: "user string representation",
			u:    &User{IP: "127.0.0.1", Isp: "Test ISP", Lat: "40.7128", Lon: "-74.0060"},
			want: "127.0.0.1 (Test ISP) [40.7128, -74.0060] ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.u.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

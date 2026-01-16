package speedtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpeedtest_FetchUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       *Speedtest
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.FetchUserInfo()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestFetchUserInfo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "fetch user info",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FetchUserInfo()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestSpeedtest_FetchUserInfoContext(t *testing.T) {
	tests := []struct {
		name    string
		s       *Speedtest
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			got, err := tt.s.FetchUserInfoContext(ctx)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestFetchUserInfoContext(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "fetch user info with context",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			got, err := FetchUserInfoContext(ctx)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestUser_String(t *testing.T) {
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

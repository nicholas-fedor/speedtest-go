package speedtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_outputTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		tr      outputTime
		want    []byte
		wantErr bool
	}{
		{
			name:    "marshal output time",
			tr:      outputTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)),
			want:    []byte(`"2023-01-01 12:00:00.000"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.tr.MarshalJSON()
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpeedtest_JSON(t *testing.T) {
	type args struct {
		servers Servers
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name: "json output with servers",
			s: &Speedtest{
				User: &User{},
			},
			args: args{
				servers: Servers{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.JSON(tt.args.servers)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			// Verify it's valid JSON
			assert.Contains(t, string(got), `"timestamp"`)
			assert.Contains(t, string(got), `"userInfo"`)
			assert.Contains(t, string(got), `"servers"`)
		})
	}
}

func TestSpeedtest_JSONL(t *testing.T) {
	type args struct {
		server *Server
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name: "jsonl output with server",
			s: &Speedtest{
				User: &User{},
			},
			args: args{
				server: &Server{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.JSONL(tt.args.server)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			// Verify it's valid JSON
			assert.Contains(t, string(got), `"timestamp"`)
			assert.Contains(t, string(got), `"userInfo"`)
			assert.Contains(t, string(got), `"server"`)
		})
	}
}

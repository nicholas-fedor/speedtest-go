package speedtest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_outputTime_MarshalJSON(t *testing.T) {
	t.Parallel()

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

// TestJSON checks that the report holds the given user and servers.
func TestJSON(t *testing.T) {
	t.Parallel()

	got, err := JSON(&User{IP: "203.0.113.7"}, Servers{&Server{ID: "1001"}})
	require.NoError(t, err)

	var report struct {
		Timestamp string    `json:"timestamp"`
		UserInfo  *User     `json:"userInfo"`
		Servers   []*Server `json:"servers"`
	}

	require.NoError(t, json.Unmarshal(got, &report))
	assert.NotEmpty(t, report.Timestamp)
	assert.Equal(t, "203.0.113.7", report.UserInfo.IP)
	require.Len(t, report.Servers, 1)
	assert.Equal(t, "1001", report.Servers[0].ID)
}

// TestJSONL checks that the line holds the given user and server, and that a missing user is null.
func TestJSONL(t *testing.T) {
	t.Parallel()

	got, err := JSONL(&User{IP: "203.0.113.7"}, &Server{ID: "1001"})
	require.NoError(t, err)
	assert.NotContains(t, string(got), "\n")
	assert.Contains(t, string(got), `"userInfo":{"ip":"203.0.113.7"`)
	assert.Contains(
		t,
		string(got),
		`"server":{"url":"","lat":"","lon":"","name":"","country":"","cc":"","sponsor":"","id":"1001"`,
	)

	got, err = JSONL(nil, &Server{})
	require.NoError(t, err)
	assert.Contains(t, string(got), `"userInfo":null`)
}

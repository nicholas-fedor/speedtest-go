package speedtest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseAddr(t *testing.T) {
	type args struct {
		addr string
	}

	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{
			name:  "address without protocol",
			args:  args{addr: "localhost:8080"},
			want:  "",
			want1: "localhost:8080",
		},
		{
			name:  "http address",
			args:  args{addr: "http://localhost:8080"},
			want:  "http",
			want1: "localhost:8080",
		},
		{
			name:  "https address",
			args:  args{addr: "https://example.com:443"},
			want:  "https",
			want1: "example.com:443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, got1 := parseAddr(tt.args.addr)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestSpeedtest_NewUserConfig(t *testing.T) {
	type args struct {
		uc *UserConfig
	}

	tests := []struct {
		name string
		s    *Speedtest
		args args
	}{
		{
			name: "valid user config",
			s:    &Speedtest{Manager: NewDataManager(), doer: &http.Client{}},
			args: args{uc: &UserConfig{UserAgent: "test", MaxConnections: 4}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.s.NewUserConfig(tt.args.uc)
			// Test passes if no panic and config is set
			assert.NotNil(t, tt.s.config)
		})
	}
}

func TestSpeedtest_RoundTrip(t *testing.T) {
	type args struct {
		req *http.Request
	}

	tests := []struct {
		name    string
		s       *Speedtest
		args    args
		wantErr bool
	}{
		{
			name:    "nil speedtest",
			s:       nil,
			args:    args{req: &http.Request{}},
			wantErr: true,
		},
		{
			name:    "nil request",
			s:       &Speedtest{},
			args:    args{req: nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.s.RoundTrip(tt.args.req)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			assert.NotNil(t, got)

			defer func() {
				_ = got.Body.Close()
			}()
		})
	}
}

func TestWithDoer(t *testing.T) {
	type args struct {
		doer *http.Client
	}

	tests := []struct {
		name string
		args args
		want Option
	}{
		{
			name: "nil http client",
			args: args{doer: nil},
			want: Option(nil), // Option is a function type, nil is expected
		},
		{
			name: "valid http client",
			args: args{doer: &http.Client{}},
			want: Option(nil), // Option is a function type, nil is expected
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := WithDoer(tt.args.doer)
			assert.NotNil(t, got) // Option function should not be nil
		})
	}
}

func TestWithUserConfig(t *testing.T) {
	type args struct {
		userConfig *UserConfig
	}

	tests := []struct {
		name string
		args args
		want Option
	}{
		{
			name: "nil user config",
			args: args{userConfig: nil},
			want: Option(nil), // Option is a function type, nil is expected
		},
		{
			name: "valid user config",
			args: args{userConfig: &UserConfig{}},
			want: Option(nil), // Option is a function type, nil is expected
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := WithUserConfig(tt.args.userConfig)
			assert.NotNil(t, got) // Option function should not be nil
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		opts []Option
	}

	tests := []struct {
		name string
		args args
		want *Speedtest
	}{
		{
			name: "no options",
			args: args{opts: nil},
			want: &Speedtest{}, // Should return a Speedtest instance
		},
		{
			name: "with options",
			args: args{opts: []Option{WithUserConfig(&UserConfig{})}},
			want: &Speedtest{}, // Should return a Speedtest instance
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := New(tt.args.opts...)
			assert.NotNil(t, got) // Should return a valid Speedtest instance
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "get version",
			want: "", // Version should be a non-empty string
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Version()
			assert.NotEmpty(t, got) // Version should be a non-empty string
		})
	}
}

package speedtest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseAddr(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	type args struct {
		doer *http.Client
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "nil http client",
			args: args{doer: nil},
		},
		{
			name: "valid http client",
			args: args{doer: &http.Client{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opt := WithDoer(tt.args.doer)
			assert.NotNil(t, opt) // Option function should not be nil

			st := &Speedtest{}
			opt(st)
			assert.Equal(t, tt.args.doer, st.doer) // Verify doer field was set
		})
	}
}

func TestWithUserConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		userConfig *UserConfig
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "valid user config",
			args: args{userConfig: &UserConfig{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opt := WithUserConfig(tt.args.userConfig)
			assert.NotNil(t, opt) // Option function should not be nil

			st := &Speedtest{Manager: NewDataManager()}
			opt(st)
			assert.Equal(t, tt.args.userConfig, st.config) // Verify config field was set
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

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
	t.Parallel()
	t.Cleanup(func() { version = "" })

	t.Run(
		"ldflag override",
		func(t *testing.T) {
			t.Parallel()

			version = "v9.9.9"

			assert.Equal(t, "9.9.9", Version())
			assert.Equal(t, "nicholas-fedor/speedtest-go 9.9.9", DefaultUserAgent())
		},
	)

	t.Run(
		"resolved non-empty",
		func(t *testing.T) {
			t.Parallel()

			version = ""
			got := Version()
			assert.NotEmpty(t, got)
			assert.NotContains(t, got, "(devel)")
		},
	)
}

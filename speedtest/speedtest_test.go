package speedtest

import (
	"net/http"
	"runtime/debug"
	"strings"
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

// TestVersion checks that Version resolves to a usable, normalized string from the real build info.
func TestVersion(t *testing.T) {
	t.Parallel()

	got := Version()

	assert.NotEmpty(t, got)
	assert.NotContains(t, got, "(devel)")
	assert.False(t, strings.HasPrefix(got, "v"), "version must not keep a leading v")
	assert.Equal(t, "nicholas-fedor/speedtest-go "+got, DefaultUserAgent())
}

// TestVersion_ldflag checks that Version passes the link-time version variable to resolveVersion.
//
// It writes the package-level version, so it must not run in parallel. Serial top-level tests finish before any
// parallel test resumes, and the cleanup restores the variable before then.
//
//nolint:paralleltest // Mutates the package-level version variable.
func TestVersion_ldflag(t *testing.T) {
	original := version

	t.Cleanup(func() { version = original })

	version = "v9.9.9"

	assert.Equal(t, "9.9.9", Version())
	assert.Equal(t, "nicholas-fedor/speedtest-go 9.9.9", DefaultUserAgent())
}

// Test_resolveVersion covers each resolution path without touching the package-level version variable.
func Test_resolveVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buildInfo *debug.BuildInfo
		name      string
		ldflag    string
		want      string
		ok        bool
	}{
		{
			name:      "ldflag wins over build info",
			ldflag:    "v9.9.9",
			buildInfo: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v1.2.3"}},
			ok:        true,
			want:      "9.9.9",
		},
		{
			name:   "ldflag is trimmed",
			ldflag: "  v1.0.0 ",
			want:   "1.0.0",
		},
		{
			name: "no build info",
			want: "dev",
		},
		{
			name: "nil build info reported as available",
			ok:   true,
			want: "dev",
		},
		{
			name:      "main module version",
			buildInfo: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v1.2.3"}},
			ok:        true,
			want:      "1.2.3",
		},
		{
			name:      "main module devel falls back to dev",
			buildInfo: &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}},
			ok:        true,
			want:      "dev",
		},
		{
			name: "dependency version when imported as a library",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "v0.1.0"},
				Deps: []*debug.Module{
					{Path: "github.com/other/module", Version: "v5.0.0"},
					{Path: modulePath, Version: "v1.4.0"},
				},
			},
			ok:   true,
			want: "1.4.0",
		},
		{
			name: "unrelated main module without dependency",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "v0.1.0"},
			},
			ok:   true,
			want: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, resolveVersion(tt.ldflag, tt.buildInfo, tt.ok))
		})
	}
}

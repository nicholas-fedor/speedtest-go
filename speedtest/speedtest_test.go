package speedtest

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
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

// TestSpeedtest_NewUserConfig checks that the dialers and transport are always built, and that the source address
// and the source-bound resolver are applied only when configured.
func TestSpeedtest_NewUserConfig(t *testing.T) {
	t.Parallel()

	control := func(string, string, syscall.RawConn) error { return nil }

	tests := []struct {
		config       *UserConfig
		wantLocalIP  net.IP
		name         string
		wantResolver bool
		wantControl  bool
		wantLoopback bool
	}{
		{name: "no source", config: &UserConfig{}},
		{
			name:        "source",
			config:      &UserConfig{Source: "127.0.0.1"},
			wantLocalIP: net.IPv4(127, 0, 0, 1),
		},
		{
			name:        "source with network prefix",
			config:      &UserConfig{Source: "tcp://127.0.0.1"},
			wantLocalIP: net.IPv4(127, 0, 0, 1),
		},
		{
			name:         "source with DNS binding",
			config:       &UserConfig{Source: "127.0.0.1", DNSBindSource: true},
			wantLocalIP:  net.IPv4(127, 0, 0, 1),
			wantResolver: true,
		},
		{name: "DNS binding without source", config: &UserConfig{DNSBindSource: true}},
		{
			name:         "hostname source with DNS binding",
			config:       &UserConfig{Source: "localhost", DNSBindSource: true},
			wantResolver: true,
			wantLoopback: true,
		},
		{name: "dialer control", config: &UserConfig{DialerControl: control}, wantControl: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := New(WithUserConfig(tt.config))

			require.NotNil(t, client.tcpDialer)
			require.NotNil(t, client.ipDialer)
			require.NotNil(t, client.config.T)
			assert.NotNil(t, client.config.T.Proxy)

			switch {
			case tt.wantLoopback:
				tcpAddr, ok := client.tcpDialer.LocalAddr.(*net.TCPAddr)
				require.True(t, ok)
				assert.True(t, tcpAddr.IP.IsLoopback())
			case tt.wantLocalIP == nil:
				assert.Nil(t, client.tcpDialer.LocalAddr)
				assert.Nil(t, client.ipDialer.LocalAddr)
			default:
				tcpAddr, ok := client.tcpDialer.LocalAddr.(*net.TCPAddr)
				require.True(t, ok)
				assert.True(t, tt.wantLocalIP.Equal(tcpAddr.IP))

				ipAddr, ok := client.ipDialer.LocalAddr.(*net.IPAddr)
				require.True(t, ok)
				assert.True(t, tt.wantLocalIP.Equal(ipAddr.IP))
			}

			if tt.wantResolver {
				require.NotNil(t, client.tcpDialer.Resolver)
				assert.True(t, client.tcpDialer.Resolver.PreferGo, "cgo ignores a custom Dial")
				assert.Same(t, client.tcpDialer.Resolver, client.ipDialer.Resolver)
			} else {
				assert.Nil(t, client.tcpDialer.Resolver)
			}

			assert.Equal(t, tt.wantControl, client.tcpDialer.Control != nil)
			assert.Equal(t, tt.wantControl, client.ipDialer.Control != nil)
		})
	}
}

// Test_resolveSource checks the DNS binding IP for literal and hostname sources.
func Test_resolveSource(t *testing.T) {
	t.Parallel()

	_, _, literal := resolveSource("tcp://127.0.0.1")
	assert.True(t, net.IPv4(127, 0, 0, 1).Equal(literal), "a literal keeps its parsed IP")

	_, _, hostname := resolveSource("localhost")
	require.NotNil(t, hostname, "a hostname binds DNS to the address it resolves to")
	assert.True(t, hostname.IsLoopback())

	_, _, empty := resolveSource("")
	assert.Nil(t, empty)

	_, _, unresolved := resolveSource("bad host")
	assert.Nil(t, unresolved, "a source that does not resolve leaves DNS unbound")
}

// Test_sourceBoundResolver checks that the resolver dials DNS servers from the source IP.
func Test_sourceBoundResolver(t *testing.T) {
	t.Parallel()

	resolver := sourceBoundResolver(net.IPv4(127, 0, 0, 1))

	conn, err := resolver.Dial(context.Background(), "udp", "127.0.0.1:53")
	require.NoError(t, err, "dialing UDP sends nothing, so no DNS server is needed")

	defer func() { _ = conn.Close() }()

	local, ok := conn.LocalAddr().(*net.UDPAddr)
	require.True(t, ok)
	assert.True(t, net.IPv4(127, 0, 0, 1).Equal(local.IP))
}

// Test_localAddrFor checks the local address type chosen for each network a resolver can dial.
func Test_localAddrFor(t *testing.T) {
	t.Parallel()

	ip := net.IPv4(127, 0, 0, 1)

	assert.Equal(t, &net.UDPAddr{IP: ip}, localAddrFor("udp4", ip))
	assert.Equal(t, &net.TCPAddr{IP: ip}, localAddrFor("tcp", ip))
	assert.Nil(t, localAddrFor("unix", ip))
}

// Test_proxyFunc checks that a configured proxy is used for every request, and that an empty or unparsable proxy
// selects the same proxy as the environment.
func Test_proxyFunc(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://speedtest.invalid/",
		nil,
	)
	require.NoError(t, err)

	proxyURL, err := proxyFunc("http://127.0.0.1:3128")(req)
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:3128", proxyURL.String())

	wantURL, wantErr := http.ProxyFromEnvironment(req)

	for _, proxy := range []string{"", "http://bad host"} {
		gotURL, gotErr := proxyFunc(proxy)(req)

		assert.Equal(t, wantURL, gotURL, "proxy %q should fall back to the environment", proxy)
		assert.Equal(t, wantErr, gotErr, "proxy %q should fall back to the environment", proxy)
	}
}

// TestNew_LeavesProcessDefaults checks that building clients never modifies the process-wide HTTP client or DNS
// resolver, which every other user of the process shares.
func TestNew_LeavesProcessDefaults(t *testing.T) {
	t.Parallel()

	transport := http.DefaultClient.Transport
	dial := reflect.ValueOf(net.DefaultResolver.Dial).Pointer()

	client := New(WithUserConfig(&UserConfig{
		Source:        "127.0.0.1",
		DNSBindSource: true,
		Proxy:         "http://127.0.0.1:3128",
	}))

	assert.NotSame(t, http.DefaultClient, client.doer)
	assert.Equal(t, transport, http.DefaultClient.Transport)
	assert.Equal(t, dial, reflect.ValueOf(net.DefaultResolver.Dial).Pointer())
}

// TestSpeedtest_UserAgent checks that requests carry the configured User-Agent exactly once, or the default.
func TestSpeedtest_UserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{name: "configured", userAgent: "speedtest-go-test/1.0", want: "speedtest-go-test/1.0"},
		{name: "default", want: DefaultUserAgent()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			client := New(WithUserConfig(&UserConfig{BaseURL: api.URL(), UserAgent: tt.userAgent}))

			_, err := client.FetchUserInfoContext(context.Background())
			require.NoError(t, err)

			requests := api.Requests()
			require.Len(t, requests, 1)
			assert.Equal(t, []string{tt.want}, requests[0].Header.Values("User-Agent"))
		})
	}
}

// TestSpeedtest_Proxy checks that requests go through the configured proxy in absolute form.
func TestSpeedtest_Proxy(t *testing.T) {
	t.Parallel()

	var (
		mu   sync.Mutex
		seen []string
	)

	record := func(target string) {
		mu.Lock()
		defer mu.Unlock()

		seen = append(seen, target)
	}

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.String())

		_, _ = io.WriteString(
			w,
			`<settings><client ip="203.0.113.7" lat="1" lon="2" isp="Proxied ISP"/></settings>`,
		)
	}))
	t.Cleanup(proxy.Close)

	client := New(
		WithUserConfig(&UserConfig{BaseURL: "http://speedtest.invalid", Proxy: proxy.URL}),
	)

	user, err := client.FetchUserInfoContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Proxied ISP", user.Isp)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, []string{"http://speedtest.invalid/speedtest-config.php"}, seen)
}

// closeBody closes a response body when there is a response.
func closeBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}

// TestSpeedtest_RoundTrip checks the nil guards, and that a request is sent with the configured User-Agent without
// changing the caller's request.
func TestSpeedtest_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("nil client", func(t *testing.T) {
		t.Parallel()

		resp, err := (*Speedtest)(nil).RoundTrip(&http.Request{})
		closeBody(resp)

		require.ErrorIs(t, err, ErrClientNil)
		assert.Nil(t, resp)
	})

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()

		resp, err := (&Speedtest{}).RoundTrip(nil)
		closeBody(resp)

		require.ErrorIs(t, err, ErrRequestNil)
		assert.Nil(t, resp)
	})

	t.Run("sends a copy with the User-Agent", func(t *testing.T) {
		t.Parallel()

		api := testserver.NewAPI(t)
		client := New(WithUserConfig(&UserConfig{UserAgent: "speedtest-go-test/1.0"}))

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			api.URL()+testserver.PathLatency,
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "caller")

		resp, err := client.RoundTrip(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(
			t,
			"caller",
			req.Header.Get("User-Agent"),
			"the caller's request must not change",
		)

		requests := api.Requests()
		require.Len(t, requests, 1)
		assert.Equal(t, []string{"speedtest-go-test/1.0"}, requests[0].Header.Values("User-Agent"))
	})
}

// TestWithDoer checks that a caller's HTTP client is copied rather than modified, keeps its settings, and sends
// requests through the speedtest round tripper unless it has its own Transport.
func TestWithDoer(t *testing.T) {
	t.Parallel()

	t.Run("nil client is ignored", func(t *testing.T) {
		t.Parallel()

		client := New(WithDoer(nil))

		require.NotNil(t, client.doer)
		assert.Same(t, client, client.doer.Transport)
	})

	t.Run("client is copied", func(t *testing.T) {
		t.Parallel()

		custom := &http.Client{Timeout: 7 * time.Second}
		client := New(WithDoer(custom))

		assert.NotSame(t, custom, client.doer)
		assert.Nil(t, custom.Transport, "the caller's client must not change")
		assert.Equal(t, 7*time.Second, client.doer.Timeout)
		assert.Same(t, client, client.doer.Transport)
	})

	t.Run("own transport is kept", func(t *testing.T) {
		t.Parallel()

		transport := &http.Transport{}
		client := New(WithDoer(&http.Client{Transport: transport}))

		assert.Same(t, transport, client.doer.Transport)
	})

	for _, order := range []string{"doer first", "config first"} {
		t.Run("user agent applies with "+order, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			doer := WithDoer(&http.Client{Timeout: 7 * time.Second})
			config := WithUserConfig(
				&UserConfig{BaseURL: api.URL(), UserAgent: "speedtest-go-test/1.0"},
			)

			opts := []Option{doer, config}
			if order == "config first" {
				opts = []Option{config, doer}
			}

			_, err := New(opts...).FetchUserInfoContext(context.Background())
			require.NoError(t, err)

			requests := api.Requests()
			require.Len(t, requests, 1)
			assert.Equal(t, "speedtest-go-test/1.0", requests[0].Header.Get("User-Agent"))
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

package speedtest

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime/debug"
	"slices"
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

// newTestClient builds a client from options the test expects to be valid.
//
// Parameters:
//   - t: the test, which fails when New returns an error.
//   - opts: the options.
//
// Returns:
//   - *Client: the client.
func newTestClient(t *testing.T, opts ...Option) *Client {
	t.Helper()

	client, err := New(opts...)
	require.NoError(t, err)

	return client
}

// TestNew_Dialers checks that the dialers and transport are always built, and that the source address and the
// source-bound resolver are applied only when configured.
func TestNew_Dialers(t *testing.T) {
	t.Parallel()

	control := func(string, string, syscall.RawConn) error { return nil }

	tests := []struct {
		wantLocalIP  net.IP
		name         string
		opts         []Option
		wantResolver bool
		wantControl  bool
		wantLoopback bool
	}{
		{name: "no source"},
		{
			name:        "source",
			opts:        []Option{WithSource("127.0.0.1")},
			wantLocalIP: net.IPv4(127, 0, 0, 1),
		},
		{
			name:        "source with network prefix",
			opts:        []Option{WithSource("tcp://127.0.0.1")},
			wantLocalIP: net.IPv4(127, 0, 0, 1),
		},
		{
			name:         "source with DNS binding",
			opts:         []Option{WithSource("127.0.0.1"), WithDNSBindSource()},
			wantLocalIP:  net.IPv4(127, 0, 0, 1),
			wantResolver: true,
		},
		{name: "DNS binding without source", opts: []Option{WithDNSBindSource()}},
		{
			name:         "hostname source with DNS binding",
			opts:         []Option{WithSource("localhost"), WithDNSBindSource()},
			wantResolver: true,
			wantLoopback: true,
		},
		{name: "dialer control", opts: []Option{WithDialerControl(control)}, wantControl: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newTestClient(t, tt.opts...)

			require.NotNil(t, client.tcpDialer)
			require.NotNil(t, client.ipDialer)
			require.NotNil(t, client.transport)
			assert.NotNil(t, client.transport.Proxy)

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

// TestNew_DialerControlsChain checks that every control added with WithDialerControl runs, in the order added.
func TestNew_DialerControlsChain(t *testing.T) {
	t.Parallel()

	var calls []string

	record := func(name string) func(string, string, syscall.RawConn) error {
		return func(string, string, syscall.RawConn) error {
			calls = append(calls, name)

			return nil
		}
	}

	client := newTestClient(
		t,
		WithDialerControl(record("first")),
		WithDialerControl(record("second")),
	)

	require.NotNil(t, client.tcpDialer.Control)
	require.NoError(t, client.tcpDialer.Control("tcp4", "127.0.0.1:80", nil))
	assert.Equal(t, []string{"first", "second"}, calls)
}

// Test_resolveSource checks the DNS binding IP for literal and hostname sources, and the error of a source that does
// not resolve.
func Test_resolveSource(t *testing.T) {
	t.Parallel()

	literal, err := resolveSource("tcp://127.0.0.1")
	require.NoError(t, err)
	assert.True(t, net.IPv4(127, 0, 0, 1).Equal(literal.sourceIP), "a literal keeps its parsed IP")
	assert.Nil(t, literal.control, "an address is not bound to a device")

	hostname, err := resolveSource("localhost")
	require.NoError(t, err)
	require.NotNil(t, hostname.sourceIP, "a hostname binds DNS to the address it resolves to")
	assert.True(t, hostname.sourceIP.IsLoopback())

	empty, err := resolveSource("")
	require.NoError(t, err)
	assert.Equal(t, sourceBinding{}, empty)

	unresolved, err := resolveSource("bad host")
	require.Error(t, err)
	assert.Equal(t, sourceBinding{}, unresolved)
}

// Test_sourceBoundResolver checks that the resolver dials DNS servers from the source IP.
func Test_sourceBoundResolver(t *testing.T) {
	t.Parallel()

	resolver := sourceBoundResolver(net.IPv4(127, 0, 0, 1), nil)

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

// TestNew_ProxySelection checks that a configured proxy is used for every request, and that without one the
// transport selects the same proxy as the environment.
func TestNew_ProxySelection(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://speedtest.invalid/",
		nil,
	)
	require.NoError(t, err)

	proxyURL, err := newTestClient(t, WithProxy("http://127.0.0.1:3128")).transport.Proxy(req)
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:3128", proxyURL.String())

	wantURL, wantErr := http.ProxyFromEnvironment(req)
	gotURL, gotErr := newTestClient(t).transport.Proxy(req)

	assert.Equal(t, wantURL, gotURL)
	assert.Equal(t, wantErr, gotErr)
}

// TestNew_LeavesProcessDefaults checks that building clients never modifies the process-wide HTTP client or DNS
// resolver, which every other user of the process shares.
func TestNew_LeavesProcessDefaults(t *testing.T) {
	t.Parallel()

	transport := http.DefaultClient.Transport
	dial := reflect.ValueOf(net.DefaultResolver.Dial).Pointer()

	client := newTestClient(
		t,
		WithSource("127.0.0.1"),
		WithDNSBindSource(),
		WithProxy("http://127.0.0.1:3128"),
	)

	assert.NotSame(t, http.DefaultClient, client.doer)
	assert.Equal(t, transport, http.DefaultClient.Transport)
	assert.Equal(t, dial, reflect.ValueOf(net.DefaultResolver.Dial).Pointer())
}

// TestClient_UserAgent checks that requests carry the configured User-Agent exactly once, or the default.
func TestClient_UserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		opts []Option
	}{
		{
			name: "configured",
			opts: []Option{WithUserAgent("speedtest-go-test/1.0")},
			want: "speedtest-go-test/1.0",
		},
		{name: "default", want: DefaultUserAgent()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := testserver.NewAPI(t)
			client := newTestClient(t, append([]Option{WithBaseURL(api.URL())}, tt.opts...)...)

			_, err := client.FetchUserInfoContext(context.Background())
			require.NoError(t, err)

			requests := api.Requests()
			require.Len(t, requests, 1)
			assert.Equal(t, []string{tt.want}, requests[0].Header.Values("User-Agent"))
		})
	}
}

// proxyRecorder starts an HTTP proxy that answers every request with user info and records each target URL and
// User-Agent.
//
// Parameters:
//   - t: the test, which closes the proxy when it ends.
//
// Returns:
//   - string: the proxy URL.
//   - func() []string: returns the targets requested so far, without their queries, each followed by its
//     User-Agent after a space.
func proxyRecorder(t *testing.T) (string, func() []string) {
	t.Helper()

	var (
		mu   sync.Mutex
		seen []string
	)

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The query is left out because it carries a random cache-busting value.
		target := *r.URL
		target.RawQuery = ""

		mu.Lock()
		defer mu.Unlock()

		seen = append(seen, target.String()+" "+r.Header.Get("User-Agent"))

		_, _ = io.WriteString(
			w,
			`<settings><client ip="203.0.113.7" lat="1" lon="2" isp="Proxied ISP"/></settings>`,
		)
	}))
	t.Cleanup(proxy.Close)

	return proxy.URL, func() []string {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(seen)
	}
}

// TestClient_Proxy checks that requests go through the configured proxy in absolute form.
func TestClient_Proxy(t *testing.T) {
	t.Parallel()

	proxy, seen := proxyRecorder(t)
	client := newTestClient(t, WithBaseURL("http://speedtest.invalid"), WithProxy(proxy))

	user, err := client.FetchUserInfoContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Proxied ISP", user.Isp)
	assert.Equal(
		t,
		[]string{"http://speedtest.invalid/speedtest-config.php " + DefaultUserAgent()},
		seen(),
	)
}

// TestUserAgentTransport checks that a request is sent with the User-Agent without changing the caller's request.
func TestUserAgentTransport(t *testing.T) {
	t.Parallel()

	api := testserver.NewAPI(t)
	transport := &userAgentTransport{
		base:      http.DefaultTransport,
		userAgent: "speedtest-go-test/1.0",
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		api.URL()+testserver.PathLatency,
		nil,
	)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "caller")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, "caller", req.Header.Get("User-Agent"), "the caller's request must not change")

	requests := api.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, []string{"speedtest-go-test/1.0"}, requests[0].Header.Values("User-Agent"))
}

// TestWithHTTPClient checks that a caller's HTTP client is copied rather than modified, keeps its settings, and
// sends requests through the client's transport unless it has its own Transport.
func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("client is copied", func(t *testing.T) {
		t.Parallel()

		custom := &http.Client{Timeout: 7 * time.Second}
		client := newTestClient(t, WithHTTPClient(custom))

		assert.NotSame(t, custom, client.doer)
		assert.Nil(t, custom.Transport, "the caller's client must not change")
		assert.Equal(t, 7*time.Second, client.doer.Timeout)

		transport, ok := client.doer.Transport.(*userAgentTransport)
		require.True(t, ok)
		assert.Same(t, client.transport, transport.base)
	})

	t.Run("own transport is kept", func(t *testing.T) {
		t.Parallel()

		transport := &http.Transport{}
		client := newTestClient(t, WithHTTPClient(&http.Client{Transport: transport}))

		assert.Same(t, transport, client.doer.Transport)
	})

	// Options are collected before the client is built, so the HTTP client gets the user agent and proxy whichever
	// order they come in.
	for _, order := range []string{"HTTP client first", "HTTP client last"} {
		t.Run("settings apply with "+order, func(t *testing.T) {
			t.Parallel()

			proxy, seen := proxyRecorder(t)
			httpClient := WithHTTPClient(&http.Client{Timeout: 7 * time.Second})
			others := []Option{
				WithBaseURL("http://speedtest.invalid"),
				WithUserAgent("speedtest-go-test/1.0"),
				WithProxy(proxy),
			}

			opts := slices.Concat([]Option{httpClient}, others)
			if order == "HTTP client last" {
				opts = slices.Concat(others, []Option{httpClient})
			}

			_, err := newTestClient(t, opts...).FetchUserInfoContext(context.Background())
			require.NoError(t, err)
			assert.Equal(
				t,
				[]string{"http://speedtest.invalid/speedtest-config.php speedtest-go-test/1.0"},
				seen(),
			)
		})
	}
}

// TestNew checks that New builds a client from default settings, and reports every invalid option at once.
func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		client := newTestClient(t)

		assert.Equal(t, DefaultUserAgent(), client.settings.userAgent)
		assert.Equal(t, DefaultBaseURL, client.settings.baseURL)
		assert.Equal(t, HTTP, client.settings.pingMode)
		assert.Equal(t, HTTPTest, client.settings.testMode)
		assert.IsType(t, noopLogger{}, client.log())
	})

	t.Run("invalid options", func(t *testing.T) {
		t.Parallel()

		client, err := New(WithUserAgent(""), nil, WithConnections(0), WithUserAgent("valid"))

		require.ErrorIs(t, err, ErrInvalidOption)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "the user agent is empty")
		assert.Contains(t, err.Error(), "the option is nil")
		assert.Contains(t, err.Error(), "connections must be at least 1")
	})
}

// recordingLogger records the messages of a client's Logger.
type recordingLogger struct {
	// mu guards messages.
	mu sync.Mutex
	// messages are the messages logged so far.
	messages []string
}

// Debug records the message.
func (l *recordingLogger) Debug(_ context.Context, msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.messages = append(l.messages, msg)
}

// TestWithLogger checks that the client and its data manager send debug messages to the logger.
func TestWithLogger(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	api := testserver.NewAPI(t)
	client := newTestClient(t, WithBaseURL(api.URL()), WithLogger(logger))

	_, err := client.FetchUserInfoContext(context.Background())
	require.NoError(t, err)

	dataManager, ok := client.Manager.(*DataManager)
	require.True(t, ok)
	assert.Same(t, logger, dataManager.log())

	logger.mu.Lock()
	defer logger.mu.Unlock()

	assert.Equal(t, []string{"client settings", "retrieving user info"}, logger.messages)
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

// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOptions_Invalid checks that each option rejects unusable values with an error wrapping ErrInvalidOption, and
// wraps the underlying sentinel where there is one.
func TestOptions_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		option  Option
		wantErr error
		name    string
	}{
		{name: "nil HTTP client", option: WithHTTPClient(nil)},
		{name: "empty user agent", option: WithUserAgent("")},
		{name: "proxy without scheme", option: WithProxy("127.0.0.1:3128")},
		{name: "proxy with unknown scheme", option: WithProxy("ftp://127.0.0.1:3128")},
		{name: "proxy without host", option: WithProxy("http://")},
		{name: "proxy that does not parse", option: WithProxy("http://bad host")},
		{name: "empty base URL", option: WithBaseURL(""), wantErr: ErrInvalidBaseURL},
		{
			name:    "base URL with another scheme",
			option:  WithBaseURL("ftp://mirror.example"),
			wantErr: ErrInvalidBaseURL,
		},
		{name: "empty source", option: WithSource("")},
		{name: "source that does not resolve", option: WithSource("bad host")},
		{name: "DSCP above 63", option: WithDSCP(64), wantErr: ErrInvalidDSCP},
		{name: "negative DSCP", option: WithDSCP(-1), wantErr: ErrInvalidDSCP},
		{name: "nil dialer control", option: WithDialerControl(nil)},
		{name: "unknown ping mode", option: WithPingMode(Proto(9))},
		{name: "unknown protocol", option: WithProtocol(TestMode(9))},
		{name: "no connections", option: WithConnections(0)},
		{name: "download size", option: WithTransferSizes(800, 0), wantErr: ErrInvalidTransferSize},
		{name: "upload size", option: WithTransferSizes(0, 2000), wantErr: ErrInvalidTransferSize},
		{name: "zero test duration", option: WithTestDuration(0)},
		{name: "negative test duration", option: WithTestDuration(-time.Second)},
		{name: "nil location", option: WithLocation(nil)},
		{name: "empty search", option: WithSearch("")},
		{name: "nil logger", option: WithLogger(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(tt.option)

			require.ErrorIs(t, err, ErrInvalidOption)
			assert.Nil(t, client)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

// TestWithProxy_RedactsPassword checks that a rejected proxy URL is reported without its password.
func TestWithProxy_RedactsPassword(t *testing.T) {
	t.Parallel()

	_, err := New(WithProxy("ftp://user:secret@127.0.0.1:3128"))

	require.ErrorIs(t, err, ErrInvalidOption)
	assert.NotContains(t, err.Error(), "secret")
}

// TestWithDSCP_Unsupported checks that a DSCP value is rejected where sockets cannot be marked.
func TestWithDSCP_Unsupported(t *testing.T) {
	t.Parallel()

	_, err := New(WithDSCP(46))

	if dscpSupported {
		require.NoError(t, err)

		return
	}

	require.ErrorIs(t, err, ErrInvalidOption)
	require.ErrorIs(t, err, ErrDSCPUnsupported)
}

// TestOptions_Settings checks that valid options store their values in the client's settings.
func TestOptions_Settings(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	location := &Location{Name: "osaka", Lat: 34.69, Lon: 135.5}

	client := newTestClient(
		t,
		WithUserAgent("agent/1.0"),
		WithProxy("socks5://127.0.0.1:1080"),
		WithBaseURL("https://mirror.example/speedtest"),
		WithSource("127.0.0.1"),
		WithDNSBindSource(),
		WithPingMode(ICMP),
		WithProtocol(TCPTest),
		WithConnections(3),
		WithTransferSizes(2000, 300),
		WithTestDuration(5*time.Second),
		WithLocation(location),
		WithSearch("fiber"),
		WithLogger(logger),
	)

	assert.Equal(t, "agent/1.0", client.settings.userAgent)
	assert.Equal(t, "socks5://127.0.0.1:1080", client.settings.proxy.String())
	assert.Equal(t, "https://mirror.example/speedtest", client.settings.baseURL)
	assert.Equal(t, "127.0.0.1", client.settings.sourceName)
	assert.True(t, client.settings.dnsBindSource)
	assert.Equal(t, ICMP, client.settings.pingMode)
	assert.Equal(t, TCPTest, client.settings.testMode)
	assert.Equal(t, 2000, client.settings.downloadSize)
	assert.Equal(t, 300, client.settings.uploadSize)
	assert.Equal(t, "fiber", client.settings.search)
	assert.Same(t, logger, client.log())

	require.NotNil(t, client.settings.location)
	assert.Equal(t, *location, *client.settings.location)
	assert.NotSame(t, location, client.settings.location, "the location is copied")

	dataManager, ok := client.Manager.(*DataManager)
	require.True(t, ok)
	assert.Equal(t, 3, dataManager.nThread)
	assert.Equal(t, 5*time.Second, dataManager.captureTime)
}

// TestOptions_LaterReplacesEarlier checks that a later option replaces the value an earlier one set.
func TestOptions_LaterReplacesEarlier(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, WithUserAgent("first/1.0"), WithUserAgent("second/1.0"))

	assert.Equal(t, "second/1.0", client.settings.userAgent)
}

// TestWithHTTPClient_Reused checks that one option can build several clients, each with its own copy of the
// caller's HTTP client, without changing the caller's client.
func TestWithHTTPClient_Reused(t *testing.T) {
	t.Parallel()

	custom := &http.Client{Timeout: time.Second}
	option := WithHTTPClient(custom)

	first := newTestClient(t, option)
	second := newTestClient(t, option)

	assert.NotSame(t, first.doer, second.doer)
	assert.NotSame(t, first.doer.Transport, second.doer.Transport)
	assert.Nil(t, custom.Transport, "the caller's client must not change")
}

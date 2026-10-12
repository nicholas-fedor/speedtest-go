// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build linux

package speedtest

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBindToDevice checks that a socket bound to the loopback device reaches a loopback server, and that binding to
// a device that does not exist fails the dial, which shows the option is applied.
func TestBindToDevice(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	address := strings.TrimPrefix(server.URL, "http://")

	bound := &net.Dialer{Control: bindToDevice(loopbackInterface(t))}

	conn, err := bound.DialContext(context.Background(), "tcp", address)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	missing := &net.Dialer{Control: bindToDevice("no-such-interface0")}

	_, err = missing.DialContext(context.Background(), "tcp", address)
	require.ErrorIs(t, err, syscall.ENODEV)
}

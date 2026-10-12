// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
)

// defaultTCPPort is the port speedtest.net servers serve their TCP protocol on.
const defaultTCPPort = "8080"

// errNoTCPHost is returned for a server with neither a host nor a URL to reach its TCP service at.
var errNoTCPHost = errors.New("server has no host for TCP tests")

// downloadRequestFor returns the download request function for the client's test mode.
//
// Returns:
//   - downloadFunc: a TCP request in [TCPTest] mode, and an HTTP request otherwise.
func (s *Server) downloadRequestFor() downloadFunc {
	if s.usesTCP() {
		return tcpDownloadRequest
	}

	return downloadRequest
}

// uploadRequestFor returns the upload request function for the client's test mode.
//
// Returns:
//   - uploadFunc: a TCP request in [TCPTest] mode, and an HTTP request otherwise.
func (s *Server) uploadRequestFor() uploadFunc {
	if s.usesTCP() {
		return tcpUploadRequest
	}

	return uploadRequest
}

// usesTCP reports whether the server's tests run over the TCP protocol.
func (s *Server) usesTCP() bool {
	return s.Context != nil && s.Context.config.TestMode == TCPTest
}

// tcpHost returns the host:port of a server's TCP service.
//
// The server's Host field usually carries the port already. Without a port, or without a Host, the port defaults to
// 8080, and a missing Host falls back to the hostname of the server's URL.
//
// Parameters:
//   - server: the server.
//
// Returns:
//   - string: the host and port.
//   - error: an error wrapping [errNoTCPHost] when neither field names a host, or the URL parse error.
func tcpHost(server *Server) (string, error) {
	host := server.Host
	if host == "" {
		parsed, err := url.Parse(server.URL)
		if err != nil {
			return "", fmt.Errorf("failed to parse server URL for a TCP test: %w", err)
		}

		host = parsed.Host
	}

	if host == "" {
		return "", fmt.Errorf("%w: server %q", errNoTCPHost, server.ID)
	}

	_, _, err := net.SplitHostPort(host)
	if err == nil {
		return host, nil
	}

	return net.JoinHostPort(strings.Trim(host, "[]"), defaultTCPPort), nil
}

// connectTCP opens a connection to a server's TCP service.
//
// Parameters:
//   - ctx: cancellation for the dial.
//   - server: the server, whose client supplies the dialer.
//
// Returns:
//   - *transport.Client: the connected client. Disconnect it when done.
//   - error: the host, client, or dial error.
func connectTCP(ctx context.Context, server *Server) (*transport.Client, error) {
	host, err := tcpHost(server)
	if err != nil {
		return nil, err
	}

	client, err := transport.NewClient(server.Context.tcpDialer)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP client: %w", err)
	}

	err = client.Connect(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for a TCP test: %w", err)
	}

	return client, nil
}

// tcpDownloadRequest downloads one block over the TCP protocol, counting each byte as it arrives.
//
// Parameters:
//   - ctx: cancellation for the request.
//   - server: the server.
//   - writer: the index into the download sizes.
//
// Returns:
//   - error: the connection, handshake, or transfer error.
func tcpDownloadRequest(ctx context.Context, server *Server, writer int) error {
	if server == nil {
		return ErrServerNil
	}

	if server.Context == nil {
		return ErrUninitializedManager
	}

	client, err := connectTCP(ctx, server)
	if err != nil {
		return err
	}

	defer func() { _ = client.Disconnect() }()

	_, err = client.VersionContext(ctx)
	if err != nil {
		return fmt.Errorf("TCP handshake failed: %w", err)
	}

	size := int64(dlSizes[writer]) * 1000

	err = client.Download(ctx, size, server.Context.NewChunk().DownloadHandler)
	if err != nil {
		return fmt.Errorf("TCP download failed: %w", err)
	}

	return nil
}

// tcpUploadRequest uploads one block over the TCP protocol and counts the payload bytes the server confirms.
//
// The server's count includes the command line and the final newline, which are not test data, so that overhead is
// subtracted before the bytes are counted.
//
// Parameters:
//   - ctx: cancellation for the request.
//   - server: the server.
//   - writer: the index into the upload sizes.
//
// Returns:
//   - error: the connection, transfer, or confirmation error.
func tcpUploadRequest(ctx context.Context, server *Server, writer int) error {
	if server == nil {
		return ErrServerNil
	}

	if server.Context == nil {
		return ErrUninitializedManager
	}

	size := int64(ulSizes[writer]) * 1000

	payload, err := transport.UploadPayloadSize(size)
	if err != nil {
		return fmt.Errorf("invalid TCP upload size: %w", err)
	}

	client, err := connectTCP(ctx, server)
	if err != nil {
		return err
	}

	defer func() { _ = client.Disconnect() }()

	confirmed, err := client.Upload(ctx, size, server.Context.NewChunk().UploadHandler(payload))
	if err != nil {
		return fmt.Errorf("TCP upload failed: %w", err)
	}

	overhead := size - payload
	if confirmed < overhead {
		return fmt.Errorf(
			"%w: confirmed %d bytes, less than the %d byte command",
			transport.ErrInvalidResponse,
			confirmed,
			overhead,
		)
	}

	server.Context.AddTotalUpload(confirmed - overhead)

	return nil
}

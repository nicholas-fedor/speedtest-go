package transport

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/testserver"
)

func TestNewPacketLossSender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		uuid   string
		dialer *net.Dialer
	}{
		{
			name:   "valid uuid",
			uuid:   "test-uuid",
			dialer: &net.Dialer{},
		},
		{
			name:   "nil dialer",
			uuid:   "test-uuid",
			dialer: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewPacketLossSender(tt.uuid, tt.dialer)
			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, strings.ToUpper(tt.uuid), got.ID)
			assert.Equal(t, tt.dialer, got.dialer)
			assert.NotEmpty(t, got.raw)
			assert.Contains(t, string(got.raw), "LOSS")
		})
	}
}

func TestPacketLossSender_Connect(t *testing.T) {
	t.Parallel()

	lc := &net.ListenConfig{}
	listener, err := lc.ListenPacket(context.Background(), "udp", "localhost:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	dialer := &net.Dialer{}
	ps, err := NewPacketLossSender("test-uuid", dialer)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("successful connect", func(t *testing.T) {
		t.Parallel()

		err := ps.Connect(ctx, listener.LocalAddr().String())
		require.NoError(t, err)
		assert.NotNil(t, ps.conn)
		assert.Equal(t, listener.LocalAddr().String(), ps.host)
	})

	t.Run("connect to invalid host", func(t *testing.T) {
		t.Parallel()

		ps2, err := NewPacketLossSender("test-uuid", dialer)
		require.NoError(t, err)
		err = ps2.Connect(ctx, "invalid:99999")
		require.Error(t, err)
	})
}

func TestPacketLossSender_Send(t *testing.T) {
	t.Parallel()

	lc := &net.ListenConfig{}
	listener, err := lc.ListenPacket(context.Background(), "udp", "localhost:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	dialer := &net.Dialer{}
	ps, err := NewPacketLossSender("test-uuid", dialer)
	require.NoError(t, err)

	ctx := context.Background()
	err = ps.Connect(ctx, listener.LocalAddr().String())
	require.NoError(t, err)

	buf := make([]byte, 1024)

	t.Run("send packet", func(t *testing.T) {
		t.Parallel()

		err := ps.Send(42)
		require.NoError(t, err)

		n, _, err := listener.ReadFrom(buf)
		require.NoError(t, err)
		assert.Contains(t, string(buf[:n]), "42")
		assert.Contains(t, string(buf[:n]), "LOSS")
	})
}

func Test_generateUUID(t *testing.T) {
	t.Parallel()

	got, err := generateUUID()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.Len(t, got, 36) // UUID format
}

// TestPacketLossSender_Close checks that Close releases the socket, is safe before Connect and when repeated, and
// that sending afterwards reports no connection instead of panicking.
func TestPacketLossSender_Close(t *testing.T) {
	t.Parallel()

	srv := testserver.NewTCPServer(t)

	sender, err := NewPacketLossSender("0f8fad5b-d9cb-469f-a165-70867728950e", &net.Dialer{})
	require.NoError(t, err)

	require.NoError(t, sender.Close(), "closing before Connect is a no-op")
	require.NoError(t, sender.Connect(context.Background(), srv.Addr()))
	require.NoError(t, sender.Send(0))
	require.NoError(t, sender.Close())
	require.NoError(t, sender.Close(), "a second Close is a no-op")

	require.ErrorIs(t, sender.Send(1), ErrEmptyConn)
}

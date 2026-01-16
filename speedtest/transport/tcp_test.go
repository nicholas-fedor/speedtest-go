package transport

import (
	"bufio"
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pingFormat(t *testing.T) {
	tests := []struct {
		name    string
		locTime int64
		want    []byte
	}{
		{
			name:    "zero time",
			locTime: 0,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '0'},
		},
		{
			name:    "positive time",
			locTime: 123,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '1', '2', '3'},
		},
		{
			name:    "negative time",
			locTime: -456,
			want:    []byte{0x50, 0x49, 0x4e, 0x47, 0x20, '-', '4', '5', '6'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pingFormat(tt.locTime)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		dialer *net.Dialer
	}{
		{
			name:   "nil dialer",
			dialer: nil,
		},
		{
			name:   "with dialer",
			dialer: &net.Dialer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(tt.dialer)
			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotEmpty(t, got.ID())
			assert.Equal(t, tt.dialer, got.dialer)
		})
	}
}

func TestClient_ID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "empty id",
			id:   "",
			want: "",
		},
		{
			name: "test id",
			id:   "test-uuid",
			want: "test-uuid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				id: tt.id,
			}
			assert.Equal(t, tt.want, client.ID())
		})
	}
}

func TestClient_Connect(t *testing.T) {
	t.Parallel()

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	dialer := &net.Dialer{}
	client, err := NewClient(dialer)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("successful connect", func(t *testing.T) {
		t.Parallel()

		err := client.Connect(ctx, listener.Addr().String())
		require.NoError(t, err)
		assert.NotNil(t, client.conn)
		assert.Equal(t, listener.Addr().String(), client.host)
		assert.NotNil(t, client.reader)
	})

	t.Run("connect to invalid host", func(t *testing.T) {
		t.Parallel()

		client2, err := NewClient(dialer)
		require.NoError(t, err)
		err = client2.Connect(ctx, "invalid:99999")
		require.Error(t, err)
	})
}

func TestClient_Disconnect(t *testing.T) {
	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	done := make(chan bool, 1)

	go func() {
		buf := make([]byte, 10)
		_, _ = client2.Read(buf) // consume the quit message

		done <- true
	}()

	err := client.Disconnect()
	require.NoError(t, err)
	assert.Nil(t, client.conn)
	assert.Nil(t, client.reader)

	<-done // wait for read
}

func TestClient_Write(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	t.Cleanup(func() { _ = client2.Close() })

	client := &Client{
		conn: client1,
	}

	t.Run("successful write", func(t *testing.T) {
		t.Parallel()

		done := make(chan string, 1)

		go func() {
			buf := make([]byte, 20)

			n, _ := client2.Read(buf)
			done <- string(buf[:n])
		}()

		err := client.Write([]byte("test data"))
		require.NoError(t, err)

		received := <-done
		assert.Equal(t, "test data\n", received)
	})

	t.Run("write with nil conn", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		err := client.Write([]byte("test"))
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})

	t.Run("write with nil conn", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		err := client.Write([]byte("test"))
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

func TestClient_Read(t *testing.T) {
	t.Parallel()

	client1, client2 := net.Pipe()

	t.Cleanup(func() { _ = client2.Close() })

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	t.Run("successful read", func(t *testing.T) {
		t.Parallel()

		done := make(chan error, 1)

		go func() {
			_, err := client2.Write([]byte("response\n"))
			done <- err
		}()

		got, err := client.Read()
		require.NoError(t, err)
		assert.Equal(t, []byte("response\n"), got)

		assert.NoError(t, <-done)
	})

	t.Run("read with nil conn", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		_, err := client.Read()
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

func TestClient_Version(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		writeHI  bool
		response string
		want     string
	}{
		{
			name:     "cached version",
			version:  "1.2.3",
			writeHI:  false,
			response: "",
			want:     "1.2.3",
		},
		{
			name:     "fetch version",
			version:  "",
			writeHI:  true,
			response: "HI 1.2.3\n",
			want:     ".3",
		},
		{
			name:     "unknown version",
			version:  "",
			writeHI:  true,
			response: "HI\n",
			want:     "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client1, client2 := net.Pipe()

			defer func() { _ = client2.Close() }()

			client := &Client{
				conn:    client1,
				reader:  bufio.NewReader(client1),
				version: tt.version,
			}

			if tt.writeHI {
				go func() {
					buf := make([]byte, 10)
					_, _ = client2.Read(buf) // consume HI\n
					_, _ = client2.Write([]byte(tt.response))
				}()
			}

			got := client.Version()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_PingContext(t *testing.T) {
	t.Parallel()

	t.Run("no connection", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		_, err := client.PingContext(context.Background())
		require.Error(t, err)
		assert.Equal(t, ErrEmptyConn, err)
	})
}

func TestClient_InitPacketLoss(t *testing.T) {
	client := &Client{
		id: "test-id",
	}

	// Test with no conn, should return error
	err := client.InitPacketLoss()
	assert.Error(t, err)
}

func TestPLoss_String(t *testing.T) {
	tests := []struct {
		name string
		p    PLoss
		want string
	}{
		{
			name: "zero sent",
			p:    PLoss{Sent: 0, Dup: 0, Max: 0},
			want: "Packet Loss: N/A",
		},
		{
			name: "normal loss",
			p:    PLoss{Sent: 90, Dup: 5, Max: 100},
			want: "Packet Loss: 15.84% (Sent: 90/Dup: 5/Max: 100)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestPLoss_Loss(t *testing.T) {
	tests := []struct {
		name string
		p    PLoss
		want float64
	}{
		{
			name: "zero sent",
			p:    PLoss{Sent: 0, Dup: 0, Max: 0},
			want: -1,
		},
		{
			name: "normal loss",
			p:    PLoss{Sent: 90, Dup: 5, Max: 100},
			want: 0.15841584158415845,
		},
		{
			name: "no loss",
			p:    PLoss{Sent: 100, Dup: 0, Max: 100},
			want: 0.00990099009900991,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, tt.p.Loss(), 1e-9)
		})
	}
}

func TestPLoss_LossPercent(t *testing.T) {
	tests := []struct {
		name string
		p    PLoss
		want float64
	}{
		{
			name: "zero sent",
			p:    PLoss{Sent: 0, Dup: 0, Max: 0},
			want: -1,
		},
		{
			name: "normal loss",
			p:    PLoss{Sent: 90, Dup: 5, Max: 100},
			want: 15.841584158415845,
		},
		{
			name: "no loss",
			p:    PLoss{Sent: 100, Dup: 0, Max: 100},
			want: 0.990099009900991,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, tt.p.LossPercent(), 1e-9)
		})
	}
}

func TestClient_PacketLoss(t *testing.T) {
	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	// Simulate server response
	go func() {
		buf := make([]byte, 10)
		_, _ = client2.Read(buf) // read PLOSS\n
		_, _ = client2.Write([]byte("PLOSS 90 5 100\n"))
	}()

	got, err := client.PacketLoss()
	require.NoError(t, err)
	assert.Equal(t, &PLoss{Sent: 90, Dup: 5, Max: 100}, got)
}

func TestClient_Download(t *testing.T) {
	client := &Client{}

	assert.Panics(t, func() { client.Download() })
}

func TestClient_Upload(t *testing.T) {
	client := &Client{}

	assert.Panics(t, func() { client.Upload() })
}

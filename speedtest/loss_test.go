package speedtest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/showwin/speedtest-go/speedtest/transport"
)

func TestNewPacketLossAnalyzer(t *testing.T) {
	type args struct {
		options *PacketLossAnalyzerOptions
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "with nil options",
			args: args{options: nil},
		},
		{
			name: "with custom options",
			args: args{options: &PacketLossAnalyzerOptions{
				SamplingDuration: time.Second * 10,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewPacketLossAnalyzer(tt.args.options)
			assert.NotNil(t, got)
			assert.NotNil(t, got.options)

			if tt.args.options != nil && tt.args.options.SamplingDuration != 0 {
				assert.Equal(t, time.Second*10, got.options.SamplingDuration)
			}
		})
	}
}

func TestPacketLossAnalyzer_RunMulti(t *testing.T) {
	type args struct {
		hosts []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty hosts",
			args:    args{hosts: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid host",
			args:    args{hosts: []string{"invalid"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)

			got, err := pla.RunMulti(tt.args.hosts)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestPacketLossAnalyzer_RunMultiWithContext(t *testing.T) {
	type args struct {
		hosts []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty hosts with context",
			args:    args{hosts: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid host with context",
			args:    args{hosts: []string{"invalid"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx := context.Background()

			got, err := pla.RunMultiWithContext(ctx, tt.args.hosts)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestPacketLossAnalyzer_Run(t *testing.T) {
	type args struct {
		host     string
		callback func(packetLoss *transport.PLoss)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid host",
			args: args{
				host:     "invalid",
				callback: func(_ *transport.PLoss) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)

			err := pla.Run(tt.args.host, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPacketLossAnalyzer_RunWithContext(t *testing.T) {
	type args struct {
		host     string
		callback func(packetLoss *transport.PLoss)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid host with context",
			args: args{
				host:     "invalid",
				callback: func(_ *transport.PLoss) {},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx := context.Background()

			err := pla.RunWithContext(ctx, tt.args.host, tt.args.callback)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPacketLossAnalyzer_loopSampler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loop sampler with cancelled context",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := pla.loopSampler(ctx, nil, func(_ *transport.PLoss) {})
			// Should return nil since context is cancelled
			assert.NoError(t, err)
		})
	}
}

func TestPacketLossAnalyzer_loopSender(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loop sender with cancelled context",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pla := NewPacketLossAnalyzer(nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()                 // Cancel immediately
			pla.loopSender(ctx, nil) // Should return immediately since context is cancelled
		})
	}
}

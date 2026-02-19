package echo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

func TestNew(t *testing.T) {
	type args struct {
		server   *speedtest.Server
		interval time.Duration
	}

	tests := []struct {
		name string
		args args
		want *AccompanyEcho
	}{
		{
			name: "create accompany echo",
			args: args{
				server:   &speedtest.Server{ID: "1"},
				interval: time.Second,
			},
			want: &AccompanyEcho{
				server:   &speedtest.Server{ID: "1"},
				interval: time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := New(tt.args.server, tt.args.interval)
			assert.NotNil(t, got)
			assert.Equal(t, tt.args.server, got.server)
			assert.Equal(t, tt.args.interval, got.interval)
		})
	}
}

func TestAccompanyEcho_Run(t *testing.T) {
	tests := []struct {
		name string
		ae   *AccompanyEcho
	}{
		{
			name: "run accompany echo",
			ae:   New(&speedtest.Server{ID: "1"}, time.Millisecond*100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.ae.Run() })
			time.Sleep(10 * time.Millisecond) // let it start
			assert.NotPanics(t, func() { tt.ae.Stop() })
		})
	}
}

func TestAccompanyEcho_Stop(t *testing.T) {
	tests := []struct {
		name string
		ae   *AccompanyEcho
	}{
		{
			name: "stop accompany echo",
			ae:   New(&speedtest.Server{ID: "1"}, time.Millisecond*100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { tt.ae.Run() })
			time.Sleep(10 * time.Millisecond)
			assert.NotPanics(t, func() { tt.ae.Stop() })
		})
	}
}

func TestAccompanyEcho_CurrentLatency(t *testing.T) {
	tests := []struct {
		name string
		ae   *AccompanyEcho
		want int64
	}{
		{
			name: "initial latency",
			ae:   New(&speedtest.Server{ID: "1"}, time.Millisecond*100),
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.ae.CurrentLatency()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAccompanyEcho_Latencies(t *testing.T) {
	tests := []struct {
		name string
		ae   *AccompanyEcho
		want []int64
	}{
		{
			name: "initial latencies",
			ae:   New(&speedtest.Server{ID: "1"}, time.Millisecond*100),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.ae.Latencies()
			assert.Equal(t, tt.want, got)
		})
	}
}

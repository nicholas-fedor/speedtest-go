// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// pingStub is a [Tester] whose pings answer at once with a fixed latency, failing every second one when fail is set.
// Its other methods are not used.
type pingStub struct {
	Tester

	// pings counts the pings sent.
	pings atomic.Int32
	// latency is the round trip of each ping.
	latency time.Duration
	// fail makes every second ping fail.
	fail bool
}

// Ping answers with the stub's latency, or fails every second ping when fail is set.
func (p *pingStub) Ping(
	context.Context,
	*speedtest.Server,
	...speedtest.PingOption,
) (speedtest.PingStats, error) {
	count := p.pings.Add(1)
	if p.fail && count%2 == 0 {
		return speedtest.PingStats{}, errors.New("timeout")
	}

	return speedtest.PingStats{Latency: p.latency}, nil
}

// TestLoadedLatency checks that the loaded latency pings once per interval until its context ends, keeps the round
// trips of the pings that answered, and reports the latest one while it runs.
func TestLoadedLatency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fail bool
		want int
	}{
		{name: "every ping answers", fail: false, want: 3},
		{name: "failed pings are left out", fail: true, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				stub := &pingStub{Tester: nil, latency: 7 * time.Millisecond, fail: tt.fail}

				ctx, cancel := context.WithCancel(t.Context())

				loaded := startLoadedLatency(ctx, stub, &speedtest.Server{}, time.Second)

				synctest.Wait()
				assert.Zero(t, loaded.current(), "no round trip before the first interval")

				time.Sleep(3500 * time.Millisecond)
				synctest.Wait()

				assert.Equal(t, 7*time.Millisecond, loaded.current())

				cancel()

				latencies := loaded.wait()
				assert.Len(t, latencies, tt.want)
				assert.EqualValues(t, 3, stub.pings.Load(), "one ping per interval")
			})
		})
	}
}

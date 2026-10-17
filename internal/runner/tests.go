// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// Packet loss measurement time limits.
const (
	// packetLossWait is how long a server without transfer tests gets for its packet loss measurement.
	packetLossWait = 30 * time.Second
	// packetLossLimit bounds the packet loss measurement while transfer tests run, which end it sooner.
	packetLossLimit = 40 * time.Second
)

// testServer measures one server: latency, then download and upload with packet loss measured in the background.
//
// Parameters:
//   - ctx: ends the tests.
//   - server: the server.
//   - pool: the servers a multi-server test spreads across.
//
// Returns:
//   - speedtest.Result: the server's measurements.
//   - error: the error of a test, or the context's error when it ended the tests.
func (r *Runner) testServer(
	ctx context.Context,
	server *speedtest.Server,
	pool speedtest.Servers,
) (speedtest.Result, error) {
	result := speedtest.Result{
		Server:     server,
		Ping:       nil,
		Download:   nil,
		Upload:     nil,
		PacketLoss: nil,
	}

	r.report(ServerStarted{Server: server})
	r.report(PingStarted{})

	stats, err := r.tester.Ping(
		ctx,
		server,
		speedtest.WithLatencyCallback(func(latency time.Duration) {
			r.report(LatencyMeasured{Latency: latency})
		}),
	)
	if err != nil {
		return result, fmt.Errorf("failed to measure the latency to server %s: %w", server.ID, err)
	}

	result.Ping = &stats
	r.report(PingFinished{Stats: stats})

	transfers := !r.settings.NoDownload || !r.settings.NoUpload

	// Transfer tests end the measurement when they finish. Without them, the measurement gets a fixed time.
	limit := packetLossWait
	if transfers {
		limit = packetLossLimit
	}

	lossCtx, stopLoss := context.WithTimeout(ctx, limit)

	var (
		loss     speedtest.PacketLossStats
		lossDone sync.WaitGroup
	)

	r.report(PacketLossStarted{})

	// A failed measurement keeps the samples it saw, and the result shows none when it saw nothing.
	lossDone.Go(func() { loss, _ = r.tester.PacketLoss(lossCtx, server) })

	defer func() {
		stopLoss()
		lossDone.Wait()
	}()

	if !r.settings.NoDownload {
		result.Download, err = r.transfer(ctx, Download, server, pool)
		if err != nil {
			return result, err
		}
	}

	if !r.settings.NoUpload {
		result.Upload, err = r.transfer(ctx, Upload, server, pool)
		if err != nil {
			return result, err
		}
	}

	if transfers {
		stopLoss()
	}

	lossDone.Wait()

	// Packet loss ends quietly with the context, so an interrupted run must not report it as measured.
	err = ctx.Err()
	if err != nil {
		return result, fmt.Errorf("speed test interrupted: %w", err)
	}

	// A server that answered no packet loss sample measured nothing, which is not the same as no loss.
	if loss.Sent > 0 {
		result.PacketLoss = &loss
	}

	r.report(PacketLossFinished{Stats: loss})
	r.report(ServerFinished{Result: result})

	return result, nil
}

// transfer runs one download or upload test while measuring the latency under its load.
//
// Parameters:
//   - ctx: ends the test.
//   - direction: whether the test downloads or uploads.
//   - server: the server, the main one of a multi-server test.
//   - pool: the servers a multi-server test spreads across.
//
// Returns:
//   - *speedtest.Transfer: the measurement.
//   - error: the test's error.
func (r *Runner) transfer(
	ctx context.Context,
	direction Direction,
	server *speedtest.Server,
	pool speedtest.Servers,
) (*speedtest.Transfer, error) {
	r.report(TransferStarted{Direction: direction})

	latencyCtx, stopLatency := context.WithCancel(ctx)
	loaded := startLoadedLatency(latencyCtx, r.tester, server, loadedLatencyInterval)

	option := speedtest.WithProgressCallback(func(progress speedtest.TransferProgress) {
		r.report(TransferProgress{
			Direction: direction,
			Rate:      progress.Rate,
			Latency:   loaded.current(),
			Elapsed:   progress.Elapsed,
			Remaining: progress.Remaining,
		})
	})

	var (
		transfer speedtest.Transfer
		err      error
	)

	switch {
	case r.settings.Multi && direction == Download:
		transfer, err = r.tester.DownloadMulti(ctx, server, pool, option)
	case r.settings.Multi:
		transfer, err = r.tester.UploadMulti(ctx, server, pool, option)
	case direction == Download:
		transfer, err = r.tester.Download(ctx, server, option)
	default:
		transfer, err = r.tester.Upload(ctx, server, option)
	}

	stopLatency()

	latencies := loaded.wait()

	if err != nil {
		return nil, fmt.Errorf("failed to measure the %s speed: %w", direction, err)
	}

	r.report(
		TransferFinished{
			Latency:   latencyStats(latencies),
			Transfer:  transfer,
			Direction: direction,
		},
	)

	return &transfer, nil
}

// latencyStats summarizes round trips the way [speedtest.Client.Ping] does.
//
// Parameters:
//   - latencies: the round trips.
//
// Returns:
//   - speedtest.PingStats: the mean, standard deviation, minimum, and maximum, all 0 without round trips. Duration
//     is 0, since the round trips were measured alongside another test.
func latencyStats(latencies []time.Duration) speedtest.PingStats {
	values := make([]int64, 0, len(latencies))
	for _, latency := range latencies {
		values = append(values, int64(latency))
	}

	mean, _, deviation, minimum, maximum := speedtest.StandardDeviation(values)

	return speedtest.PingStats{
		Latencies: latencies,
		Latency:   time.Duration(mean),
		Jitter:    time.Duration(deviation),
		Min:       time.Duration(minimum),
		Max:       time.Duration(maximum),
		Duration:  0,
	}
}

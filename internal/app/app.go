// Package app provides application logic for speedtest operations.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/internal/echo"
	"github.com/showwin/speedtest-go/internal/output"
	"github.com/showwin/speedtest-go/internal/parser"
	"github.com/showwin/speedtest-go/internal/task"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/showwin/speedtest-go/speedtest/transport"
)

const (
	bytesToMB                 = 1000 * 1000
	nanoToMilli               = 1000000
	packetLossAnalyzerTimeout = 40 * time.Second
	echoInterval              = 500 * time.Millisecond
	sleepAfterTests           = 30 * time.Second
)

// setupSpeedtestClient creates and configures the speedtest client.
func setupSpeedtestClient(cfg Config) *speedtest.Speedtest {
	return speedtest.New(speedtest.WithUserConfig(
		&speedtest.UserConfig{
			UserAgent:      cfg.UserAgent,
			Proxy:          cfg.Proxy,
			Source:         cfg.Source,
			DNSBindSource:  cfg.DNSBindSource,
			Debug:          cfg.Debug,
			PingMode:       parser.ParseProto(cfg.PingMode),
			SavingMode:     cfg.SavingMode,
			MaxConnections: cfg.Thread,
			CityFlag:       cfg.City,
			LocationFlag:   cfg.Location,
			Keyword:        cfg.Search,
		}))
}

// retrieveServers fetches and selects the target servers.
func retrieveServers(
	speedtestClient *speedtest.Speedtest, cfg Config, taskManager *task.Manager,
) (speedtest.Servers, speedtest.Servers) {
	var (
		err     error
		servers speedtest.Servers
		targets speedtest.Servers
	)

	taskManager.Run("Retrieving Servers", func(task *task.Task) {
		switch {
		case len(cfg.CustomURL) > 0:
			var target *speedtest.Server

			target, err = speedtestClient.CustomServer(cfg.CustomURL)
			task.CheckError(err)

			targets = []*speedtest.Server{target}

			task.Println("Skip: Using Custom Server")
		case len(cfg.ServerIDs) > 0:
			type fetchResult struct {
				server *speedtest.Server
				err    error
			}

			results := make(chan fetchResult, len(cfg.ServerIDs))
			for _, id := range cfg.ServerIDs {
				go func(id int) {
					serverPtr, errFetch := speedtestClient.FetchServerByID(strconv.Itoa(id))
					results <- fetchResult{server: serverPtr, err: errFetch}
				}(id)
			}

			for range cfg.ServerIDs {
				res := <-results
				if res.err != nil {
					continue // Silently Skip all ids that actually don't exist.
				}

				targets = append(targets, res.server)
			}

			task.CheckError(err)
			task.Printf("Found %d Specified Public Server(s)", len(targets))
		default:
			servers, err = speedtestClient.FetchServers()
			task.CheckError(err)
			task.Printf("Found %d Public Servers", len(servers))
			targets, err = servers.FindServer(cfg.ServerIDs)
			task.CheckError(err)
		}

		task.Complete()
	})

	return servers, targets
}

// runTest executes the bandwidth test based on configuration.
func runTest(
	server *speedtest.Server,
	cfg Config,
	task *task.Task,
	isDownload bool,
	servers speedtest.Servers,
) {
	switch {
	case cfg.Multi && isDownload:
		task.CheckError(server.MultiDownloadTestContext(context.Background(), servers))
	case cfg.Multi && !isDownload:
		task.CheckError(server.MultiUploadTestContext(context.Background(), servers))
	case isDownload:
		task.CheckError(server.DownloadTest())
	default:
		task.CheckError(server.UploadTest())
	}
}

// setCallback sets the appropriate callback for download or upload.
func setCallback(
	client *speedtest.Speedtest,
	isDownload bool,
	accEcho *echo.AccompanyEcho,
	task *task.Task,
) {
	if isDownload {
		client.SetCallbackDownload(func(downRate speedtest.ByteRate) {
			updateWithLatency(task, downRate, accEcho, "Download")
		})
	} else {
		client.SetCallbackUpload(func(upRate speedtest.ByteRate) {
			updateWithLatency(task, upRate, accEcho, "Upload")
		})
	}
}

// updateWithLatency updates the task with rate and latency information.
func updateWithLatency(
	task *task.Task,
	rate speedtest.ByteRate,
	accEcho *echo.AccompanyEcho,
	prefix string,
) {
	lc := accEcho.CurrentLatency()
	if lc == 0 {
		task.Updatef("%s: %s (Latency: --)", prefix, rate)
	} else {
		task.Updatef("%s: %s (Latency: %dms)", prefix, rate, lc/nanoToMilli)
	}
}

// runBandwidthTest performs download or upload bandwidth tests.
func runBandwidthTest(
	isDownload bool, server *speedtest.Server, cfg Config, taskManager *task.Manager,
	accEcho *echo.AccompanyEcho, speedtestClient *speedtest.Speedtest, servers speedtest.Servers,
) {
	taskName := "Download"
	trigger := !cfg.NoDownload

	if !isDownload {
		taskName = "Upload"
		trigger = !cfg.NoUpload
	}

	taskManager.RunWithTrigger(trigger, taskName, func(task *task.Task) {
		accEcho.Run()

		setCallback(speedtestClient, isDownload, accEcho, task)

		runTest(server, cfg, task, isDownload, servers)

		accEcho.Stop()
		mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())

		speed := server.DLSpeed

		total := float64(server.Context.GetTotalDownload())
		if !isDownload {
			speed = server.ULSpeed
			total = float64(server.Context.GetTotalUpload())
		}

		task.Printf(
			"%s: %s (Used: %.2fMB) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)",
			taskName,
			speed,
			total/bytesToMB,
			mean/nanoToMilli,
			std/nanoToMilli,
			minL/nanoToMilli,
			maxL/nanoToMilli,
		)
		task.Complete()
	})
}

// runServerTests performs tests for a single server.
func runServerTests(
	server *speedtest.Server, cfg Config, taskManager *task.Manager,
	speedtestClient *speedtest.Speedtest, servers speedtest.Servers,
) {
	if !cfg.JSONOutput && !cfg.JSONLOutput {
		log.Println()
	}

	taskManager.Println("Test Server: " + server.String())
	taskManager.Run("Latency: --", func(task *task.Task) {
		task.CheckError(server.PingTest(func(latency time.Duration) {
			task.Updatef("Latency: %v", latency)
		}))
		task.Printf("Latency: %v Jitter: %v Min: %v Max: %v",
			server.Latency, server.Jitter, server.MinLatency, server.MaxLatency)
		task.Complete()
	})

	// create a packet loss analyzer
	analyzer := speedtest.NewPacketLossAnalyzer(&speedtest.PacketLossAnalyzerOptions{
		SourceInterface: cfg.Source,
	})

	blocker := sync.WaitGroup{}
	packetLossAnalyzerCtx, packetLossAnalyzerCancel := context.WithTimeout(
		context.Background(),
		packetLossAnalyzerTimeout,
	)

	taskManager.Run("Packet Loss Analyzer", func(task *task.Task) {
		blocker.Go(func() {
			err := analyzer.RunWithContext(
				packetLossAnalyzerCtx,
				server.Host,
				func(packetLoss *transport.PLoss) {
					server.PacketLoss = *packetLoss
				},
			)
			if errors.Is(err, transport.ErrUnsupported) {
				packetLossAnalyzerCancel()
			}
		})

		task.Println("Packet Loss Analyzer: Running in background (<= 30 Secs)")
		task.Complete()
	})

	// create accompany Echo
	accEcho := echo.New(server, echoInterval)

	runBandwidthTest(true, server, cfg, taskManager, accEcho, speedtestClient, servers)
	runBandwidthTest(false, server, cfg, taskManager, accEcho, speedtestClient, servers)

	if cfg.NoUpload && cfg.NoDownload {
		time.Sleep(sleepAfterTests)
	}

	packetLossAnalyzerCancel()
	blocker.Wait()

	if !cfg.JSONOutput && !cfg.JSONLOutput {
		taskManager.Println(server.PacketLoss.String())
	}

	taskManager.Reset()
	speedtestClient.Reset()
}

// runTests performs the actual speed tests on the selected servers.
func runTests(
	speedtestClient *speedtest.Speedtest, targets, servers speedtest.Servers,
	cfg Config, taskManager *task.Manager,
) error {
	// 3. test each selected server with ping, download and upload.
	for _, server := range targets {
		runServerTests(server, cfg, taskManager, speedtestClient, servers)
	}

	taskManager.Stop()

	if cfg.JSONOutput {
		json, errMarshal := speedtestClient.JSON(targets)
		if errMarshal != nil {
			return fmt.Errorf("failed to marshal JSON: %w", errMarshal)
		}

		log.Print(string(json))
	} else if cfg.JSONLOutput {
		for _, server := range targets {
			json, errMarshal := speedtestClient.JSONL(server)
			if errMarshal != nil {
				return fmt.Errorf("failed to marshal JSONL: %w", errMarshal)
			}

			log.Println(string(json))
		}
	}

	return nil
}

// RunSpeedtest executes the full speedtest on selected servers.
func RunSpeedtest(cfg Config) error {
	setupConfig(cfg)

	speedtestClient := setupSpeedtestClient(cfg)

	output.AppInfo(cfg.JSONOutput, cfg.JSONLOutput)

	// retrieving user information
	taskManager := task.NewManager(cfg.JSONOutput || cfg.JSONLOutput, cfg.UnixOutput)
	taskManager.AsyncRun("Retrieving User Information", func(t *task.Task) {
		u, err := speedtestClient.FetchUserInfo()
		t.CheckError(err)
		t.Printf("ISP: %s", u.String())
		t.Complete()
	})

	servers, targets := retrieveServers(speedtestClient, cfg, taskManager)

	taskManager.Reset()

	return runTests(speedtestClient, targets, servers, cfg, taskManager)
}

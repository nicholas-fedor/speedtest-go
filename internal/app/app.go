// Package app provides application logic for speedtest operations.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/echo"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/task"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest/transport"
)

const (
	bytesToMB                 = 1000 * 1000
	nanoToMilli               = 1000000
	packetLossAnalyzerTimeout = 40 * time.Second
	echoInterval              = 500 * time.Millisecond
	sleepAfterTests           = 30 * time.Second
	percent                   = 100
)

// setupSpeedtestClient creates and configures the speedtest client.
func setupSpeedtestClient(cfg config.Config) *speedtest.Speedtest {
	return speedtest.New(speedtest.WithUserConfig(
		&speedtest.UserConfig{
			UserAgent:      cfg.UserAgent,
			Proxy:          cfg.Proxy,
			Source:         cfg.Source,
			DNSBindSource:  cfg.DNSBindSource,
			Debug:          cfg.Debug,
			PingMode:       cfg.PingMode,
			TestMode:       cfg.TestMode,
			SavingMode:     cfg.SavingMode,
			MaxConnections: cfg.Thread,
			CityFlag:       cfg.City,
			LocationFlag:   cfg.Location,
			Keyword:        cfg.Search,
			BaseURL:        cfg.BaseURL,
			DSCP:           cfg.DSCP,
			DownloadSize:   cfg.DownloadSize,
			UploadSize:     cfg.UploadSize,
		},
	))
}

// retrieveServers fetches and selects the target servers. It returns the pool for --multi and the targets.
// Servers named with --custom-url or --server are their own pool.
func retrieveServers(
	speedtestClient *speedtest.Speedtest,
	cfg config.Config,
	taskManager *task.Manager,
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
			servers = targets

			task.Println("Skip: Using Custom Server")
		case len(cfg.ServerIDs) > 0:
			targets, err = fetchServersByID(speedtestClient, cfg.ServerIDs)
			task.CheckError(err)

			servers = targets
			task.Printf("Found %d Specified Public Server(s)", len(targets))
		default:
			servers, err = speedtestClient.FetchServers()
			task.CheckError(err)

			if len(cfg.CountryCodes) > 0 {
				var codes []string

				servers, codes, err = filterByCountry(speedtestClient, cfg, servers)
				task.CheckError(err)
				task.Printf(
					"Found %d Public Servers in %s",
					len(servers),
					strings.Join(codes, ", "),
				)
			} else {
				task.Printf("Found %d Public Servers", len(servers))
			}

			targets, err = servers.FindServer(cfg.ServerIDs)
			task.CheckError(err)
		}

		task.Complete()
	})

	return servers, targets
}

// fetchServersByID looks up each server ID concurrently.
//
// Parameters:
//   - speedtestClient: the client that performs the lookups.
//   - ids: the server IDs from --server.
//
// Returns:
//   - speedtest.Servers: the servers in the order of ids.
//   - error: an error wrapping [speedtest.ErrServerNotFound] that lists every unknown ID, joined with any other
//     lookup errors.
func fetchServersByID(speedtestClient *speedtest.Speedtest, ids []int) (speedtest.Servers, error) {
	found := make(speedtest.Servers, len(ids))
	errs := make([]error, len(ids))

	var wg sync.WaitGroup

	for index, id := range ids {
		wg.Go(func() {
			found[index], errs[index] = speedtestClient.FetchServerByID(strconv.Itoa(id))
		})
	}

	wg.Wait()

	var (
		missing []string
		failed  []error
	)

	for index, serverID := range ids {
		switch err := errs[index]; {
		case err == nil:
		case errors.Is(err, speedtest.ErrServerNotFound):
			missing = append(missing, strconv.Itoa(serverID))
		default:
			failed = append(failed, fmt.Errorf("server %d: %w", serverID, err))
		}
	}

	if len(missing) > 0 {
		failed = append(
			[]error{
				fmt.Errorf(
					"%w for --server %s",
					speedtest.ErrServerNotFound,
					strings.Join(missing, ", "),
				),
			},
			failed...,
		)
	}

	if len(failed) > 0 {
		return nil, errors.Join(failed...)
	}

	return found, nil
}

// runTest executes the bandwidth test based on configuration.
func runTest(
	server *speedtest.Server,
	cfg config.Config,
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
	currentLatency := accEcho.CurrentLatency()
	if currentLatency == 0 {
		task.Updatef("%s: %s (Latency: --)",
			prefix, rate,
		)
	} else {
		task.Updatef("%s: %s (Latency: %dms)",
			prefix, rate, currentLatency/nanoToMilli,
		)
	}
}

// runBandwidthTest performs download or upload bandwidth tests.
func runBandwidthTest(
	isDownload bool, server *speedtest.Server, cfg config.Config, taskManager *task.Manager,
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
		used := fmt.Sprintf("Used: %.2fMB", total/bytesToMB)

		if !isDownload {
			speed = server.ULSpeed
			total = float64(server.Context.GetTotalUpload())
			used = fmt.Sprintf(
				"Used: %.2fMB Confirmed: %.1f%%",
				total/bytesToMB,
				server.Context.GetUploadConfirmationRatio()*percent,
			)
		}

		task.Printf(
			"%s: %s (%s) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)",
			taskName,
			speed,
			used,
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
	server *speedtest.Server, cfg config.Config, taskManager *task.Manager,
	speedtestClient *speedtest.Speedtest, servers speedtest.Servers,
) {
	if !cfg.JSONOutput && !cfg.JSONLOutput {
		fmt.Fprintln(os.Stdout)
	}

	taskManager.Println("Test Server: " + server.String())
	taskManager.Run("Latency: --", func(task *task.Task) {
		task.CheckError(server.PingTest(func(latency time.Duration) {
			task.Updatef("Latency: %v", latency)
		}))
		task.Printf("Latency: %v Jitter: %v Min: %v Max: %v",
			server.Latency,
			server.Jitter,
			server.MinLatency,
			server.MaxLatency,
		)
		task.Complete()
	})

	// create a packet loss analyzer
	analyzer := speedtestClient.NewPacketLossAnalyzer(nil)

	blocker := sync.WaitGroup{}
	packetLossAnalyzerCtx, packetLossAnalyzerCancel := context.WithTimeout(
		context.Background(),
		packetLossAnalyzerTimeout,
	)

	var (
		lastValidLoss transport.PLoss
		hasValidLoss  bool
	)

	taskManager.Run("Packet Loss Analyzer", func(task *task.Task) {
		blocker.Go(func() {
			err := analyzer.RunWithContext(
				packetLossAnalyzerCtx,
				server.Host,
				func(packetLoss *transport.PLoss) {
					if packetLoss.Sent != 0 {
						lastValidLoss = *packetLoss
						hasValidLoss = true
					}
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

	runBandwidthTest(
		true,
		server,
		cfg,
		taskManager,
		accEcho,
		speedtestClient,
		servers,
	)
	runBandwidthTest(
		false,
		server,
		cfg,
		taskManager,
		accEcho,
		speedtestClient,
		servers,
	)

	if cfg.NoUpload && cfg.NoDownload {
		time.Sleep(sleepAfterTests)
	}

	packetLossAnalyzerCancel()
	blocker.Wait()

	if hasValidLoss {
		server.PacketLoss = lastValidLoss
	}

	if !cfg.JSONOutput && !cfg.JSONLOutput {
		taskManager.Println(server.PacketLoss.String())
	}

	taskManager.Reset()
	speedtestClient.Reset()
}

// runTests performs the actual speed tests on the selected servers.
func runTests(
	speedtestClient *speedtest.Speedtest, targets, servers speedtest.Servers,
	cfg config.Config, taskManager *task.Manager,
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

		fmt.Fprintf(os.Stdout, "%s", json)
	} else if cfg.JSONLOutput {
		for _, server := range targets {
			json, errMarshal := speedtestClient.JSONL(server)
			if errMarshal != nil {
				return fmt.Errorf(
					"failed to marshal JSONL: %w",
					errMarshal,
				)
			}

			fmt.Fprintf(os.Stdout, "%s\n", json)
		}
	}

	return nil
}

// RunSpeedtest executes the full speedtest on selected servers.
func RunSpeedtest(cfg config.Config) error {
	// Retrieve the configuration.
	cfg = config.Setup(cfg)

	// Create the speedtest client.
	speedtestClient := setupSpeedtestClient(cfg)

	// Print the application information.
	output.AppInfo(cfg.JSONOutput, cfg.JSONLOutput)

	// Create a new task manager.
	taskManager := task.NewManager(
		cfg.JSONOutput || cfg.JSONLOutput,
		cfg.UnixOutput,
	)

	// Retrieve user information
	taskManager.AsyncRun(
		"Retrieving User Information",
		func(t *task.Task) {
			u, err := speedtestClient.FetchUserInfo()
			t.CheckError(err)
			t.Printf("ISP: %s", u.String())
			t.Complete()
		},
	)

	// Fetch and select the target server.
	servers, targets := retrieveServers(
		speedtestClient,
		cfg, taskManager,
	)

	// Reset the task manager.
	taskManager.Reset()

	// Perform the test.
	return runTests(speedtestClient, targets, servers, cfg, taskManager)
}

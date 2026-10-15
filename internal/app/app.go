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
	// userInfoTimeout bounds the user info request, which the reports wait for.
	userInfoTimeout = 30 * time.Second
)

// setupSpeedtestClient creates the speedtest client from the settings.
//
// Parameters:
//   - cfg: the validated settings.
//
// Returns:
//   - *speedtest.Client: the client.
//   - error: an error wrapping [config.ErrInvalidSetting] when the client rejects a setting, such as a --source that
//     does not resolve.
func setupSpeedtestClient(cfg config.Config) (*speedtest.Client, error) {
	opts, err := clientOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", config.ErrInvalidSetting, err)
	}

	client, err := speedtest.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", config.ErrInvalidSetting, err)
	}

	return client, nil
}

// clientOptions translates the settings into speedtest client options.
//
// Settings left at their zero values add no option, so the client keeps its defaults.
//
// Parameters:
//   - cfg: the validated settings.
//
// Returns:
//   - []speedtest.Option: the options.
//   - error: the error of a --city or --location value that names no location.
func clientOptions(cfg config.Config) ([]speedtest.Option, error) {
	opts := []speedtest.Option{
		speedtest.WithPingMode(cfg.PingMode),
		speedtest.WithProtocol(cfg.TestMode),
		speedtest.WithDSCP(cfg.DSCP),
		speedtest.WithTransferSizes(cfg.DownloadSize, cfg.UploadSize),
	}

	stringOptions := []struct {
		option func(string) speedtest.Option
		value  string
	}{
		{option: speedtest.WithUserAgent, value: cfg.UserAgent},
		{option: speedtest.WithProxy, value: cfg.Proxy},
		{option: speedtest.WithSource, value: cfg.Source},
		{option: speedtest.WithSearch, value: cfg.Search},
		{option: speedtest.WithBaseURL, value: cfg.BaseURL},
	}

	for _, setting := range stringOptions {
		if setting.value != "" {
			opts = append(opts, setting.option(setting.value))
		}
	}

	if cfg.DNSBindSource {
		opts = append(opts, speedtest.WithDNSBindSource())
	}

	switch {
	case cfg.SavingMode:
		opts = append(opts, speedtest.WithConnections(1))
	case cfg.Thread > 0:
		opts = append(opts, speedtest.WithConnections(cfg.Thread))
	default:
		// Without a connection count, the client uses one connection per CPU.
	}

	location, err := settingsLocation(cfg)
	if err != nil {
		return nil, err
	}

	if location != nil {
		opts = append(opts, speedtest.WithLocation(location))
	}

	if cfg.Debug {
		opts = append(opts, speedtest.WithLogger(newDebugLogger(os.Stderr)))
	}

	return opts, nil
}

// settingsLocation returns the location that --city or --location names.
//
// Parameters:
//   - cfg: the validated settings.
//
// Returns:
//   - *speedtest.Location: the location, or nil when neither setting is used.
//   - error: the error of a value that names no location.
func settingsLocation(cfg config.Config) (*speedtest.Location, error) {
	switch {
	case cfg.City != "":
		location, err := speedtest.GetLocation(cfg.City)
		if err != nil {
			return nil, fmt.Errorf("--city %q: %w", cfg.City, err)
		}

		return location, nil
	case cfg.Location != "":
		location, err := speedtest.ParseLocation("", cfg.Location)
		if err != nil {
			return nil, fmt.Errorf("--location %q: %w", cfg.Location, err)
		}

		return location, nil
	default:
		return nil, nil //nolint:nilnil // No location setting means no location and no error.
	}
}

// retrieveServers fetches and selects the target servers. It returns the pool for --multi and the targets.
// Servers named with --custom-url or --server are their own pool.
func retrieveServers(
	speedtestClient *speedtest.Client,
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
func fetchServersByID(speedtestClient *speedtest.Client, ids []int) (speedtest.Servers, error) {
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

// runTransfer runs the download or upload test that the settings select.
//
// Parameters:
//   - ctx: ends the test early.
//   - client: the client.
//   - server: the server.
//   - servers: the pool for --multi.
//   - cfg: the settings.
//   - isDownload: whether the test downloads.
//   - onRate: receives the live rate.
//
// Returns:
//   - speedtest.Transfer: the measurement.
//   - error: the test's error.
func runTransfer(
	ctx context.Context,
	client *speedtest.Client,
	server *speedtest.Server,
	servers speedtest.Servers,
	cfg config.Config,
	isDownload bool,
	onRate func(rate speedtest.ByteRate),
) (speedtest.Transfer, error) {
	option := speedtest.WithRateCallback(onRate)

	var (
		transfer speedtest.Transfer
		err      error
	)

	switch {
	case cfg.Multi && isDownload:
		transfer, err = client.DownloadMulti(ctx, server, servers, option)
	case cfg.Multi:
		transfer, err = client.UploadMulti(ctx, server, servers, option)
	case isDownload:
		transfer, err = client.Download(ctx, server, option)
	default:
		transfer, err = client.Upload(ctx, server, option)
	}

	if err != nil {
		return transfer, fmt.Errorf("failed to measure the transfer: %w", err)
	}

	return transfer, nil
}

// updateWithLatency updates the task with rate and latency information.
func updateWithLatency(
	task *task.Task,
	rate speedtest.ByteRate,
	unit speedtest.UnitType,
	accEcho *echo.AccompanyEcho,
	prefix string,
) {
	currentLatency := accEcho.CurrentLatency()
	if currentLatency == 0 {
		task.Updatef("%s: %s (Latency: --)",
			prefix, rate.Format(unit),
		)
	} else {
		task.Updatef("%s: %s (Latency: %dms)",
			prefix, rate.Format(unit), currentLatency/nanoToMilli,
		)
	}
}

// runBandwidthTest performs a download or upload test and shows its result.
//
// Returns:
//   - *speedtest.Transfer: the measurement, or nil when the settings skip the test.
func runBandwidthTest(
	isDownload bool, server *speedtest.Server, cfg config.Config, taskManager *task.Manager,
	accEcho *echo.AccompanyEcho, speedtestClient *speedtest.Client, servers speedtest.Servers,
) *speedtest.Transfer {
	taskName := "Download"
	trigger := !cfg.NoDownload

	if !isDownload {
		taskName = "Upload"
		trigger = !cfg.NoUpload
	}

	var result *speedtest.Transfer

	taskManager.RunWithTrigger(trigger, taskName, func(task *task.Task) {
		accEcho.Run()

		transfer, err := runTransfer(
			context.Background(),
			speedtestClient,
			server,
			servers,
			cfg,
			isDownload,
			func(rate speedtest.ByteRate) { updateWithLatency(task, rate, cfg.Unit, accEcho, taskName) },
		)
		task.CheckError(err)

		result = &transfer

		accEcho.Stop()
		mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())

		used := fmt.Sprintf("Used: %.2fMB", float64(transfer.Bytes)/bytesToMB)
		if !isDownload {
			used = fmt.Sprintf(
				"Used: %.2fMB Confirmed: %.1f%%",
				float64(transfer.Bytes)/bytesToMB,
				transfer.ConfirmationRatio()*percent,
			)
		}

		task.Printf(
			"%s: %s (%s) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)",
			taskName,
			transfer.Rate.Format(cfg.Unit),
			used,
			mean/nanoToMilli,
			std/nanoToMilli,
			minL/nanoToMilli,
			maxL/nanoToMilli,
		)
		task.Complete()
	})

	return result
}

// runServerTests performs the tests of one server and shows their results.
//
// Returns:
//   - speedtest.Result: the server's measurements.
func runServerTests(
	server *speedtest.Server, cfg config.Config, taskManager *task.Manager,
	speedtestClient *speedtest.Client, servers speedtest.Servers,
) speedtest.Result {
	result := speedtest.Result{
		Server:     server,
		Ping:       nil,
		Download:   nil,
		Upload:     nil,
		PacketLoss: nil,
	}

	if !cfg.JSONOutput && !cfg.JSONLOutput {
		fmt.Fprintln(os.Stdout)
	}

	taskManager.Println("Test Server: " + serverLine(server, cfg))
	taskManager.Run("Latency: --", func(task *task.Task) {
		stats, err := speedtestClient.Ping(
			context.Background(),
			server,
			speedtest.WithLatencyCallback(func(latency time.Duration) {
				task.Updatef("Latency: %v", latency)
			}),
		)
		task.CheckError(err)

		result.Ping = &stats

		task.Printf(
			"Latency: %v Jitter: %v Min: %v Max: %v",
			stats.Latency,
			stats.Jitter,
			stats.Min,
			stats.Max,
		)
		task.Complete()
	})

	blocker := sync.WaitGroup{}
	packetLossCtx, packetLossCancel := context.WithTimeout(
		context.Background(),
		packetLossAnalyzerTimeout,
	)

	var packetLoss transport.PLoss

	taskManager.Run("Packet Loss Analyzer", func(task *task.Task) {
		blocker.Go(func() {
			// A failed measurement keeps the samples it saw, and the result shows none when it saw nothing.
			packetLoss, _ = speedtestClient.PacketLoss(packetLossCtx, server)
		})

		task.Println("Packet Loss Analyzer: Running in background (<= 30 Secs)")
		task.Complete()
	})

	// create accompany Echo
	accEcho := echo.New(speedtestClient, server, echoInterval)

	result.Download = runBandwidthTest(
		true,
		server,
		cfg,
		taskManager,
		accEcho,
		speedtestClient,
		servers,
	)
	result.Upload = runBandwidthTest(
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

	packetLossCancel()
	blocker.Wait()

	// A server that answered no packet loss sample measured nothing, which is not the same as no loss.
	if packetLoss.Sent > 0 {
		result.PacketLoss = &packetLoss
	}

	if !cfg.JSONOutput && !cfg.JSONLOutput {
		taskManager.Println(packetLoss.String())
	}

	taskManager.Reset()

	return result
}

// runTests performs the actual speed tests on the selected servers.
//
// Parameters:
//   - speedtestClient: the client.
//   - targets: the servers to test.
//   - servers: the pool for --multi.
//   - cfg: the settings.
//   - taskManager: shows the progress.
//   - user: waits for and returns the caller's information for the reports.
//
// Returns:
//   - error: the error of writing a report.
func runTests(
	speedtestClient *speedtest.Client, targets, servers speedtest.Servers,
	cfg config.Config, taskManager *task.Manager, user func() *speedtest.User,
) error {
	// 3. test each selected server with ping, download and upload.
	results := make([]speedtest.Result, 0, len(targets))
	for _, server := range targets {
		results = append(
			results,
			runServerTests(server, cfg, taskManager, speedtestClient, servers),
		)
	}

	taskManager.Stop()

	now := time.Now()

	jsonReport, jsonlReport := output.JSONReport, output.JSONLReport
	if cfg.MaskISP {
		jsonReport, jsonlReport = output.MaskedJSONReport, output.MaskedJSONLReport
	}

	if cfg.JSONOutput {
		json, errMarshal := jsonReport(now, user(), results)
		if errMarshal != nil {
			return fmt.Errorf("failed to marshal JSON: %w", errMarshal)
		}

		fmt.Fprintf(os.Stdout, "%s", json)
	} else if cfg.JSONLOutput {
		for _, result := range results {
			json, errMarshal := jsonlReport(now, user(), result)
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

// reportedUser returns the user details to show, masked when --mask-isp is set.
//
// Parameters:
//   - user: the user details from speedtest.net, or nil when unknown.
//   - cfg: the settings.
//
// Returns:
//   - *speedtest.User: user, or a masked copy of it. The given user is never changed.
func reportedUser(user *speedtest.User, cfg config.Config) *speedtest.User {
	if cfg.MaskISP {
		return output.MaskUser(user)
	}

	return user
}

// serverLine describes a server, with its distance masked when --mask-isp is set.
//
// Parameters:
//   - server: the server.
//   - cfg: the settings.
//
// Returns:
//   - string: the description.
func serverLine(server *speedtest.Server, cfg config.Config) string {
	if cfg.MaskISP {
		return output.MaskedServerLine(server)
	}

	return server.String()
}

// RunSpeedtest executes the full speedtest on selected servers.
func RunSpeedtest(cfg config.Config) error {
	// Retrieve the configuration.
	cfg = config.Setup(cfg)

	// Create the speedtest client.
	speedtestClient, err := setupSpeedtestClient(cfg)
	if err != nil {
		return err
	}

	// Print the application information.
	output.AppInfo(cfg.JSONOutput, cfg.JSONLOutput)

	// Create a new task manager.
	taskManager := task.NewManager(
		cfg.JSONOutput || cfg.JSONLOutput,
		cfg.UnixOutput,
	)

	// Retrieve user information while the servers are retrieved. The reports wait for it.
	var (
		user        *speedtest.User
		userFetched sync.WaitGroup
	)

	userFetched.Add(1)
	taskManager.AsyncRun(
		"Retrieving User Information",
		func(userTask *task.Task) {
			defer userFetched.Done()

			ctx, cancel := context.WithTimeout(context.Background(), userInfoTimeout)
			defer cancel()

			u, err := speedtestClient.FetchUserInfoContext(ctx)
			userTask.CheckError(err)

			user = u

			userTask.Printf("ISP: %s", reportedUser(u, cfg).String())
			userTask.Complete()
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
	return runTests(speedtestClient, targets, servers, cfg, taskManager, func() *speedtest.User {
		userFetched.Wait()

		return user
	})
}

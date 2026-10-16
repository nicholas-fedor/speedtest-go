// Package app provides application logic for speedtest operations.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/report"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
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

// RunSpeedtest runs the speed test on the selected servers, showing its progress, and writes the JSON report when
// one is requested.
//
// Parameters:
//   - ctx: ends the run. The CLI cancels it on an interrupt.
//   - cfg: the settings.
//   - out: receives the progress and the reports.
//
// Returns:
//   - error: an error wrapping [config.ErrInvalidSetting] for a setting the client rejects, the error that ended the
//     run, which wraps the context's error when the context ended it, or the error of writing a report.
func RunSpeedtest(ctx context.Context, cfg config.Config, out io.Writer) error {
	cfg = config.Setup(cfg)

	//nolint:contextcheck // speedtest.New takes no context, and only its debug log of the settings uses one.
	client, err := setupSpeedtestClient(cfg)
	if err != nil {
		return err
	}

	if !cfg.JSONOutput && !cfg.JSONLOutput {
		output.WriteAppInfo(out)
	}

	user, results, err := runner.New(client, newReporter(cfg, out), runSettings(cfg)).Run(ctx)
	if err != nil {
		return err //nolint:wrapcheck // The runner names the lookup or test that failed.
	}

	return writeReports(out, cfg, user, results)
}

// newReporter returns the reporter for the output the settings select: nothing besides the JSON reports, one line
// per result for --unix, or live spinners.
//
// Parameters:
//   - cfg: the settings.
//   - out: receives the progress.
//
// Returns:
//   - runner.Reporter: the reporter.
func newReporter(cfg config.Config, out io.Writer) runner.Reporter {
	options := report.Options{Unit: cfg.Unit, MaskISP: cfg.MaskISP}

	switch {
	case cfg.JSONOutput || cfg.JSONLOutput:
		return report.Discard{}
	case cfg.UnixOutput:
		return report.NewLines(out, options)
	default:
		return report.NewSpinner(out, options)
	}
}

// runSettings returns the settings that choose a run's servers and tests.
//
// Parameters:
//   - cfg: the settings.
//
// Returns:
//   - runner.Settings: the run settings.
func runSettings(cfg config.Config) runner.Settings {
	return runner.Settings{
		CustomURL:    cfg.CustomURL,
		City:         cfg.City,
		ServerIDs:    cfg.ServerIDs,
		CountryCodes: cfg.CountryCodes,
		Multi:        cfg.Multi,
		NoDownload:   cfg.NoDownload,
		NoUpload:     cfg.NoUpload,
	}
}

// writeReports writes the --json or --jsonl report of a finished run, masked when --mask-isp is set.
//
// Parameters:
//   - out: receives the report.
//   - cfg: the settings.
//   - user: the caller's details.
//   - results: the results of the tested servers.
//
// Returns:
//   - error: the error of marshaling or writing a report.
func writeReports(
	out io.Writer,
	cfg config.Config,
	user *speedtest.User,
	results []speedtest.Result,
) error {
	now := time.Now()

	jsonReport, jsonlReport := output.JSONReport, output.JSONLReport
	if cfg.MaskISP {
		jsonReport, jsonlReport = output.MaskedJSONReport, output.MaskedJSONLReport
	}

	switch {
	case cfg.JSONOutput:
		data, err := jsonReport(now, user, results)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		_, err = out.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write the JSON report: %w", err)
		}
	case cfg.JSONLOutput:
		for _, result := range results {
			data, err := jsonlReport(now, user, result)
			if err != nil {
				return fmt.Errorf("failed to marshal JSONL: %w", err)
			}

			_, err = fmt.Fprintf(out, "%s\n", data)
			if err != nil {
				return fmt.Errorf("failed to write the JSONL report: %w", err)
			}
		}
	default:
		// The other outputs showed the results as they arrived.
	}

	return nil
}

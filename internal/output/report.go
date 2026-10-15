// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// reportTimeLayout is the layout of the timestamp in JSON reports.
const reportTimeLayout = "2006-01-02 15:04:05.000"

// report is the --json report of a run, with every tested server.
type report struct {
	// Timestamp is when the report was written.
	Timestamp reportTime `json:"timestamp"`
	// UserInfo is the user details, possibly masked, or nil when unknown.
	UserInfo *speedtest.User `json:"userInfo"`
	// Servers holds []speedtest.Result, or []speedtest.ResultReport without distances in a masked report.
	Servers any `json:"servers"`
}

// serverReport is one line of the --jsonl report, with one tested server.
type serverReport struct {
	// Timestamp is when the line was written.
	Timestamp reportTime `json:"timestamp"`
	// UserInfo is the user details, possibly masked, or nil when unknown.
	UserInfo *speedtest.User `json:"userInfo"`
	// Server holds a [speedtest.Result], or a [speedtest.ResultReport] without its distance in a masked report.
	Server any `json:"server"`
}

// reportTime is a report timestamp, written in [reportTimeLayout].
type reportTime time.Time

// MarshalJSON writes the timestamp as a quoted string in [reportTimeLayout].
//
// Returns:
//   - []byte: the quoted timestamp.
//   - error: always nil.
func (t reportTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(reportTimeLayout) + `"`), nil
}

// JSONReport renders the --json report of a run.
//
// Parameters:
//   - timestamp: the time of the report.
//   - user: the user details to report, possibly masked, or nil when unknown.
//   - results: the results of the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func JSONReport(
	timestamp time.Time,
	user *speedtest.User,
	results []speedtest.Result,
) ([]byte, error) {
	data, err := json.Marshal(
		report{Timestamp: reportTime(timestamp), UserInfo: user, Servers: results},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the JSON report: %w", err)
	}

	return data, nil
}

// JSONLReport renders one line of the --jsonl report, for one tested server.
//
// Parameters:
//   - timestamp: the time of the report.
//   - user: the user details to report, possibly masked, or nil when unknown.
//   - result: the result of the tested server.
//
// Returns:
//   - []byte: the JSON document, without a newline.
//   - error: the marshalling error.
func JSONLReport(
	timestamp time.Time,
	user *speedtest.User,
	result speedtest.Result,
) ([]byte, error) {
	data, err := json.Marshal(
		serverReport{Timestamp: reportTime(timestamp), UserInfo: user, Server: result},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the JSONL report: %w", err)
	}

	return data, nil
}

// MaskedJSONReport renders the --json report of a run for --mask-isp: the user's IP address and coordinates are
// masked, and each server's distance from the user is left out.
//
// Parameters:
//   - timestamp: the time of the report.
//   - user: the user details, unmasked, or nil when unknown.
//   - results: the results of the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func MaskedJSONReport(
	timestamp time.Time,
	user *speedtest.User,
	results []speedtest.Result,
) ([]byte, error) {
	masked := make([]speedtest.ResultReport, 0, len(results))
	for _, result := range results {
		masked = append(masked, maskedResult(result))
	}

	data, err := json.Marshal(
		report{Timestamp: reportTime(timestamp), UserInfo: MaskUser(user), Servers: masked},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the JSON report: %w", err)
	}

	return data, nil
}

// MaskedJSONLReport renders one line of the --jsonl report for --mask-isp, masked like [MaskedJSONReport].
//
// Parameters:
//   - timestamp: the time of the report.
//   - user: the user details, unmasked, or nil when unknown.
//   - result: the result of the tested server.
//
// Returns:
//   - []byte: the JSON document, without a newline.
//   - error: the marshalling error.
func MaskedJSONLReport(
	timestamp time.Time,
	user *speedtest.User,
	result speedtest.Result,
) ([]byte, error) {
	data, err := json.Marshal(serverReport{
		Timestamp: reportTime(timestamp),
		UserInfo:  MaskUser(user),
		Server:    maskedResult(result),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the JSONL report: %w", err)
	}

	return data, nil
}

// maskedResult returns the report form of a result without the server's distance, because distances to servers at
// known coordinates reveal where the user is.
//
// Parameters:
//   - result: the result.
//
// Returns:
//   - speedtest.ResultReport: the report form, with no distance.
func maskedResult(result speedtest.Result) speedtest.ResultReport {
	masked := result.Report()
	masked.Distance = nil

	return masked
}

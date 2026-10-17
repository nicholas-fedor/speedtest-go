// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"encoding/json"
	"fmt"
	"time"
)

// Result collects the measurements of one server. A measurement that was not taken is nil.
type Result struct {
	// Server is the tested server.
	Server *Server
	// Ping is the latency from [Client.Ping], or nil.
	Ping *PingStats
	// Download is the download test from [Client.Download] or [Client.DownloadMulti], or nil.
	Download *Transfer
	// Upload is the upload test from [Client.Upload] or [Client.UploadMulti], or nil.
	Upload *Transfer
	// PacketLoss is the packet loss from [Client.PacketLoss], or nil.
	PacketLoss *PacketLossStats
}

// ResultReport is the JSON form of a [Result]: the server's details followed by its measurements, with the field
// names of the speedtest-go JSON reports.
//
// The field order is the key order of the JSON reports, so betteralign leaves it alone.
//
//nolint:tagliatelle // Field names must match the speedtest-go JSON report format.
type ResultReport struct { // betteralign:ignore
	// URL is the server's upload URL.
	URL string `json:"url"`
	// Lat is the server's latitude.
	Lat string `json:"lat"`
	// Lon is the server's longitude.
	Lon string `json:"lon"`
	// Name is the server's city name.
	Name string `json:"name"`
	// Country is the server's country name.
	Country string `json:"country"`
	// CC is the server's country code.
	CC string `json:"cc"`
	// Sponsor is the organization that hosts the server.
	Sponsor string `json:"sponsor"`
	// ID is the server's identifier.
	ID string `json:"id"`
	// Host is the server's host and port.
	Host string `json:"host"`
	// Distance is the server's distance in kilometres, or nil to leave it out.
	Distance *float64 `json:"distance,omitempty"`
	// Latency is the mean round trip, or 0 without a ping.
	Latency time.Duration `json:"latency"`
	// MaxLatency is the longest round trip, or 0 without a ping.
	MaxLatency time.Duration `json:"maxLatency"`
	// MinLatency is the shortest round trip, or 0 without a ping.
	MinLatency time.Duration `json:"minLatency"`
	// Jitter is the standard deviation of the round trips, or 0 without a ping.
	Jitter time.Duration `json:"jitter"`
	// DLSpeed is the download rate, or 0 without a download test.
	DLSpeed ByteRate `json:"dlSpeed"`
	// ULSpeed is the upload rate, or 0 without an upload test.
	ULSpeed ByteRate `json:"ulSpeed"`
	// TestDuration is how long each measurement took.
	TestDuration TestDuration `json:"testDuration"`
	// PacketLoss is the packet loss, or zero counts without a measurement.
	PacketLoss PacketLossStats `json:"packetLoss"`
}

// TestDuration is how long each measurement of a [ResultReport] took. A measurement that was not taken is nil.
type TestDuration struct {
	// Ping is how long the ping took.
	Ping *time.Duration `json:"ping"`
	// Download is how long the download test took.
	Download *time.Duration `json:"download"`
	// Upload is how long the upload test took.
	Upload *time.Duration `json:"upload"`
	// Total is the sum of the other durations, or nil when no measurement was taken.
	Total *time.Duration `json:"total"`
}

// fullOutput is the JSON document [JSON] writes.
type fullOutput struct {
	// Timestamp is when the document was written.
	Timestamp outputTime `json:"timestamp"`
	// UserInfo is the caller's information, or nil.
	UserInfo *User `json:"userInfo"`
	// Servers are the results.
	Servers []Result `json:"servers"`
}

// singleServerOutput is the JSON line [JSONL] writes.
type singleServerOutput struct {
	// Timestamp is when the line was written.
	Timestamp outputTime `json:"timestamp"`
	// UserInfo is the caller's information, or nil.
	UserInfo *User `json:"userInfo"`
	// Server is the result.
	Server Result `json:"server"`
}

// outputTime is a timestamp, written as a quoted string with millisecond precision.
type outputTime time.Time

// Report returns the JSON form of the result.
//
// Returns:
//   - ResultReport: the server's details and its measurements.
func (r Result) Report() ResultReport {
	report := ResultReport{}

	if server := r.Server; server != nil {
		distance := server.Distance
		report.URL, report.Lat, report.Lon = server.URL, server.Lat, server.Lon
		report.Name, report.Country, report.CC = server.Name, server.Country, server.CC
		report.Sponsor, report.ID, report.Host = server.Sponsor, server.ID, server.Host
		report.Distance = &distance
	}

	if r.Ping != nil {
		report.Latency, report.MaxLatency = r.Ping.Latency, r.Ping.Max
		report.MinLatency, report.Jitter = r.Ping.Min, r.Ping.Jitter
		report.TestDuration.Ping = measuredDuration(r.Ping.Duration)
	}

	if r.Download != nil {
		report.DLSpeed = r.Download.Rate
		report.TestDuration.Download = measuredDuration(r.Download.Duration)
	}

	if r.Upload != nil {
		report.ULSpeed = r.Upload.Rate
		report.TestDuration.Upload = measuredDuration(r.Upload.Duration)
	}

	if r.PacketLoss != nil {
		report.PacketLoss = *r.PacketLoss
	}

	report.TestDuration.Total = totalDuration(
		report.TestDuration.Ping,
		report.TestDuration.Download,
		report.TestDuration.Upload,
	)

	return report
}

// MarshalJSON writes the result in its report form.
//
// Returns:
//   - []byte: the JSON object.
//   - error: the marshalling error.
func (r Result) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(r.Report())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the result: %w", err)
	}

	return data, nil
}

// MarshalJSON writes the timestamp as a quoted string with millisecond precision.
//
// Returns:
//   - []byte: the quoted timestamp.
//   - error: always nil.
func (t outputTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02 15:04:05.000") + `"`), nil
}

// JSON marshals the results of a run to JSON.
//
// Parameters:
//   - user: the caller's information from [Client.FetchUserInfo], or nil.
//   - results: the results of the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func JSON(user *User, results []Result) ([]byte, error) {
	data, err := json.Marshal(
		fullOutput{Timestamp: outputTime(time.Now()), UserInfo: user, Servers: results},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal speedtest results to JSON: %w", err)
	}

	return data, nil
}

// JSONL marshals the result of one server to one line of JSON.
//
// Parameters:
//   - user: the caller's information from [Client.FetchUserInfo], or nil.
//   - result: the result of the tested server.
//
// Returns:
//   - []byte: the JSON line, without a trailing newline.
//   - error: the marshalling error.
func JSONL(user *User, result Result) ([]byte, error) {
	data, err := json.Marshal(
		singleServerOutput{Timestamp: outputTime(time.Now()), UserInfo: user, Server: result},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server result to JSON: %w", err)
	}

	return data, nil
}

// measuredDuration returns a measurement's duration for a report.
//
// Parameters:
//   - duration: the duration.
//
// Returns:
//   - *time.Duration: the duration, or nil when it is 0, which means nothing was measured.
func measuredDuration(duration time.Duration) *time.Duration {
	if duration <= 0 {
		return nil
	}

	return &duration
}

// totalDuration returns the sum of the measured durations.
//
// Parameters:
//   - durations: the durations, nil where nothing was measured.
//
// Returns:
//   - *time.Duration: the sum, or nil when nothing was measured.
func totalDuration(durations ...*time.Duration) *time.Duration {
	var (
		total    time.Duration
		measured bool
	)

	for _, duration := range durations {
		if duration != nil {
			total += *duration
			measured = true
		}
	}

	if !measured {
		return nil
	}

	return &total
}

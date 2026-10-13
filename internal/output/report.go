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
	Timestamp reportTime      `json:"timestamp"`
	UserInfo  *speedtest.User `json:"userInfo"`
	// Servers holds [speedtest.Servers], or []maskedServer in a masked report.
	Servers any `json:"servers"`
}

// serverReport is one line of the --jsonl report, with one tested server.
type serverReport struct {
	Timestamp reportTime      `json:"timestamp"`
	UserInfo  *speedtest.User `json:"userInfo"`
	// Server holds a *[speedtest.Server], or a maskedServer in a masked report.
	Server any `json:"server"`
}

// maskedServer is a server in a masked report. Its distance from the user is left out, because distances to servers
// at known coordinates reveal where the user is. The empty Distance field takes precedence over the server's own,
// so every other field keeps its name and order.
type maskedServer struct {
	*speedtest.Server

	// Distance is always nil, which omits the server's distance.
	Distance *float64 `json:"distance,omitempty"`
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
//   - servers: the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func JSONReport(
	timestamp time.Time,
	user *speedtest.User,
	servers speedtest.Servers,
) ([]byte, error) {
	data, err := json.Marshal(
		report{Timestamp: reportTime(timestamp), UserInfo: user, Servers: servers},
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
//   - server: the tested server.
//
// Returns:
//   - []byte: the JSON document, without a newline.
//   - error: the marshalling error.
func JSONLReport(
	timestamp time.Time,
	user *speedtest.User,
	server *speedtest.Server,
) ([]byte, error) {
	data, err := json.Marshal(
		serverReport{Timestamp: reportTime(timestamp), UserInfo: user, Server: server},
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
//   - servers: the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func MaskedJSONReport(
	timestamp time.Time,
	user *speedtest.User,
	servers speedtest.Servers,
) ([]byte, error) {
	masked := make([]maskedServer, 0, len(servers))
	for _, server := range servers {
		masked = append(masked, maskedServer{Server: server, Distance: nil})
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
//   - server: the tested server.
//
// Returns:
//   - []byte: the JSON document, without a newline.
//   - error: the marshalling error.
func MaskedJSONLReport(
	timestamp time.Time,
	user *speedtest.User,
	server *speedtest.Server,
) ([]byte, error) {
	data, err := json.Marshal(serverReport{
		Timestamp: reportTime(timestamp),
		UserInfo:  MaskUser(user),
		Server:    maskedServer{Server: server, Distance: nil},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the JSONL report: %w", err)
	}

	return data, nil
}

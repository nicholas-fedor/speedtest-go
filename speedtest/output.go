package speedtest

import (
	"encoding/json"
	"fmt"
	"time"
)

type fullOutput struct {
	Timestamp outputTime `json:"timestamp"`
	UserInfo  *User      `json:"userInfo"`
	Servers   Servers    `json:"servers"`
}

type singleServerOutput struct {
	Timestamp outputTime `json:"timestamp"`
	UserInfo  *User      `json:"userInfo"`
	Server    *Server    `json:"server"`
}

type outputTime time.Time

func (t outputTime) MarshalJSON() ([]byte, error) {
	_ = t // ensure outputTime implements json.Marshaler

	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006-01-02 15:04:05.000"))

	return []byte(stamp), nil
}

// JSON marshals the speedtest results to JSON.
//
// Parameters:
//   - user: the caller's information from [Client.FetchUserInfo], or nil.
//   - servers: the tested servers.
//
// Returns:
//   - []byte: the JSON document.
//   - error: the marshalling error.
func JSON(user *User, servers Servers) ([]byte, error) {
	data, err := json.Marshal(
		fullOutput{
			Timestamp: outputTime(time.Now()),
			UserInfo:  user,
			Servers:   servers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal speedtest results to JSON: %w", err)
	}

	return data, nil
}

// JSONL marshals a single server result to one line of JSON.
//
// Parameters:
//   - user: the caller's information from [Client.FetchUserInfo], or nil.
//   - server: the tested server.
//
// Returns:
//   - []byte: the JSON line, without a trailing newline.
//   - error: the marshalling error.
func JSONL(user *User, server *Server) ([]byte, error) {
	data, err := json.Marshal(
		singleServerOutput{
			Timestamp: outputTime(time.Now()),
			UserInfo:  user,
			Server:    server,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server result to JSON: %w", err)
	}

	return data, nil
}

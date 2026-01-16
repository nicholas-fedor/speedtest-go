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
func (s *Speedtest) JSON(servers Servers) ([]byte, error) {
	data, err := json.Marshal(
		fullOutput{
			Timestamp: outputTime(time.Now()),
			UserInfo:  s.User,
			Servers:   servers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal speedtest results to JSON: %w", err)
	}

	return data, nil
}

// JSONL outputs a single server result in JSON format.
func (s *Speedtest) JSONL(server *Server) ([]byte, error) {
	data, err := json.Marshal(
		singleServerOutput{
			Timestamp: outputTime(time.Now()),
			UserInfo:  s.User,
			Server:    server,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server result to JSON: %w", err)
	}

	return data, nil
}

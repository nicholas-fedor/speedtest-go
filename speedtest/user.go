package speedtest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
)

const speedTestConfigURL = "https://www.speedtest.net/speedtest-config.php"

var (
	// ErrInstanceNil is returned when the speedtest instance is nil.
	ErrInstanceNil = errors.New("speedtest instance is nil")
	// ErrFetchUserInfo is returned when fetching user information fails.
	ErrFetchUserInfo = errors.New("failed to fetch user information")
)

// User represents information determined about the caller by speedtest.net.
type User struct {
	IP  string `json:"ip"  xml:"ip,attr"`
	Lat string `json:"lat" xml:"lat,attr"`
	Lon string `json:"lon" xml:"lon,attr"`
	Isp string `json:"isp" xml:"isp,attr"`
}

// Users for decode xml.
type Users struct {
	Users []User `xml:"client"`
}

// FetchUserInfo returns information about caller determined by speedtest.net.
func (s *Speedtest) FetchUserInfo() (*User, error) {
	return s.FetchUserInfoContext(context.Background())
}

// FetchUserInfo returns information about caller determined by speedtest.net.
func FetchUserInfo() (*User, error) {
	return defaultClient.FetchUserInfo()
}

// FetchUserInfoContext returns information about caller determined by speedtest.net, observing the given context.
func (s *Speedtest) FetchUserInfoContext(ctx context.Context) (*User, error) {
	if s == nil {
		return nil, ErrInstanceNil
	}

	dbg.Printf("Retrieving user info: %s\n", speedTestConfigURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, speedTestConfigURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	// Decode xml
	decoder := xml.NewDecoder(resp.Body)

	var users Users

	err = decoder.Decode(&users)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	if len(users.Users) == 0 {
		return nil, ErrFetchUserInfo
	}

	s.User = &users.Users[0]

	return s.User, nil
}

// FetchUserInfoContext returns information about caller determined by speedtest.net, observing the given context.
func FetchUserInfoContext(ctx context.Context) (*User, error) {
	return defaultClient.FetchUserInfoContext(ctx)
}

// String representation of User.
func (u *User) String() string {
	extInfo := ""

	return fmt.Sprintf("%s (%s) [%s, %s] %s", u.IP, u.Isp, u.Lat, u.Lon, extInfo)
}

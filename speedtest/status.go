package speedtest

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrUnexpectedStatus is returned, wrapped in a [StatusError], when a server answers with a status outside 2xx.
var ErrUnexpectedStatus = errors.New("unexpected HTTP status")

// StatusError reports an HTTP response whose status is outside 2xx.
type StatusError struct {
	// URL is the requested URL, with any password redacted.
	URL string
	// StatusCode is the HTTP status code the server returned.
	StatusCode int
}

// Error describes the status and the URL that returned it.
//
// Returns:
//   - string: the error message.
func (e *StatusError) Error() string {
	return fmt.Sprintf(
		"%s: %d %s from %s",
		ErrUnexpectedStatus,
		e.StatusCode,
		http.StatusText(e.StatusCode),
		e.URL,
	)
}

// Unwrap returns [ErrUnexpectedStatus], so callers can match any status error with [errors.Is].
//
// Returns:
//   - error: [ErrUnexpectedStatus].
func (e *StatusError) Unwrap() error {
	return ErrUnexpectedStatus
}

// checkStatus reports a response whose status is outside 2xx.
//
// Parameters:
//   - resp: the response to check.
//
// Returns:
//   - error: a *StatusError wrapping [ErrUnexpectedStatus], or nil for a 2xx status.
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	var requested string
	if resp.Request != nil && resp.Request.URL != nil {
		requested = resp.Request.URL.Redacted()
	}

	return &StatusError{URL: requested, StatusCode: resp.StatusCode}
}

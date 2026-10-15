// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

// SetCallbackDownload sets the function that receives the live download rate during each download test.
//
// Parameters:
//   - callback: receives the rate, or nil for none.
func (s *Client) SetCallbackDownload(callback func(rate ByteRate)) {
	if s == nil || s.meter == nil {
		return
	}

	s.meter.SetCallbackDownload(rateCallback(callback))
}

// SetCallbackUpload sets the function that receives the live upload rate during each upload test.
//
// Parameters:
//   - callback: receives the rate, or nil for none.
func (s *Client) SetCallbackUpload(callback func(rate ByteRate)) {
	if s == nil || s.meter == nil {
		return
	}

	s.meter.SetCallbackUpload(rateCallback(callback))
}

// GetTotalDownload returns the bytes downloaded since the client was built or last reset.
//
// Returns:
//   - int64: the downloaded bytes, or 0 for a client not built by [New].
func (s *Client) GetTotalDownload() int64 {
	if s == nil || s.meter == nil {
		return 0
	}

	return s.meter.TotalDownload()
}

// GetTotalUpload returns the upload bytes the servers confirmed since the client was built or last reset.
//
// Returns:
//   - int64: the confirmed upload bytes, or 0 for a client not built by [New].
func (s *Client) GetTotalUpload() int64 {
	if s == nil || s.meter == nil {
		return 0
	}

	return s.meter.TotalUpload()
}

// GetUploadConfirmationRatio returns the share of the upload bytes sent since the client was built or last reset
// that the servers confirmed. It counts application-level confirmations, not TCP acknowledgments.
//
// Returns:
//   - float64: the share from 0 to 1, or 0 before any bytes were sent.
func (s *Client) GetUploadConfirmationRatio() float64 {
	if s == nil || s.meter == nil {
		return 0
	}

	return s.meter.UploadConfirmationRatio()
}

// Reset clears the download and upload totals, so the next tests count from zero.
func (s *Client) Reset() {
	if s == nil || s.meter == nil {
		return
	}

	s.meter.Reset()
}

// hasMeter reports whether the server's client can measure transfers.
//
// Returns:
//   - bool: true when the server has a client built by [New].
func (s *Server) hasMeter() bool {
	return s != nil && s.Context != nil && s.Context.meter != nil
}

// rateCallback adapts a rate callback to the meter's, which reports bytes per second.
//
// Parameters:
//   - callback: the callback, or nil.
//
// Returns:
//   - func(rate float64): the adapted callback, or nil for a nil callback.
func rateCallback(callback func(rate ByteRate)) func(rate float64) {
	if callback == nil {
		return nil
	}

	return func(rate float64) { callback(ByteRate(rate)) }
}

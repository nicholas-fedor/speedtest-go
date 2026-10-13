// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Default indexes into the transfer size tables, used when no size is configured.
const (
	// defaultDownloadIndex selects 1000x1000 images for downloads.
	defaultDownloadIndex = 3
	// defaultUploadIndex selects 1000 kB uploads in single-server tests.
	defaultUploadIndex = 4
	// defaultMultiUploadIndex selects 800 kB uploads in multi-server tests.
	defaultMultiUploadIndex = 3
)

// ErrInvalidTransferSize is returned for a download or upload size that is not in its size table.
var ErrInvalidTransferSize = errors.New("invalid transfer size")

// CheckDownloadSize reports whether a value can be given to [WithTransferSizes] as the download size.
//
// Parameters:
//   - size: the download size, or 0 for the default.
//
// Returns:
//   - error: an error wrapping [ErrInvalidTransferSize] that lists the accepted sizes, or nil.
func CheckDownloadSize(size int) error {
	return checkSize(dlSizes[:], size, "download")
}

// CheckUploadSize reports whether a value can be given to [WithTransferSizes] as the upload size.
//
// Parameters:
//   - size: the upload size in kB, or 0 for the default.
//
// Returns:
//   - error: an error wrapping [ErrInvalidTransferSize] that lists the accepted sizes, or nil.
func CheckUploadSize(size int) error {
	return checkSize(ulSizes[:], size, "upload")
}

// checkSize reports whether size is 0 or a value in table.
//
// Parameters:
//   - table: the accepted sizes.
//   - size: the size to check.
//   - direction: "download" or "upload", for the error message.
//
// Returns:
//   - error: an error wrapping [ErrInvalidTransferSize] that lists the accepted sizes, or nil.
func checkSize(table []int, size int, direction string) error {
	if size == 0 || slices.Contains(table, size) {
		return nil
	}

	accepted := make([]string, len(table))
	for index, value := range table {
		accepted[index] = strconv.Itoa(value)
	}

	return fmt.Errorf("%w: %d is not a %s size, use one of %s",
		ErrInvalidTransferSize, size, direction, strings.Join(accepted, ", "))
}

// sizeIndex returns the index of a configured size in its table.
//
// Parameters:
//   - table: the accepted sizes.
//   - size: the configured size, or 0 for the default.
//   - fallback: the index to use when no valid size is configured.
//
// Returns:
//   - int: the index of size, or fallback.
func sizeIndex(table []int, size, fallback int) int {
	index := slices.Index(table, size)
	if index < 0 {
		return fallback
	}

	return index
}

// downloadIndex returns the download size index for a request.
//
// Parameters:
//   - fallback: the index for this kind of test when no size is configured.
//
// Returns:
//   - int: the index into the download sizes.
func (s *Client) downloadIndex(fallback int) int {
	return sizeIndex(dlSizes[:], s.settings.downloadSize, fallback)
}

// uploadIndex returns the upload size index for a request.
//
// Parameters:
//   - fallback: the index for this kind of test when no size is configured.
//
// Returns:
//   - int: the index into the upload sizes.
func (s *Client) uploadIndex(fallback int) int {
	return sizeIndex(ulSizes[:], s.settings.uploadSize, fallback)
}

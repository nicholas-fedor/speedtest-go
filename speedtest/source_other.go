// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build !linux

package speedtest

// bindToDevice returns no Control function, since this platform has no SO_BINDTODEVICE. A source interface still
// sets the source address.
//
// Parameters:
//   - name: the interface name, unused.
//
// Returns:
//   - dialControl: nil.
func bindToDevice(_ string) dialControl {
	return nil
}

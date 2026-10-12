// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

//go:build linux

package speedtest

import (
	"fmt"
	"syscall"
)

// bindToDevice returns a dialer Control function that binds each socket to the named network interface with
// SO_BINDTODEVICE, so its traffic leaves through that interface whatever the routing table prefers. Kernels before
// 5.7 require CAP_NET_RAW for it.
//
// Parameters:
//   - name: the interface name.
//
// Returns:
//   - dialControl: the Control function.
func bindToDevice(name string) dialControl {
	return func(_, _ string, conn syscall.RawConn) error {
		var bindErr error

		err := conn.Control(func(fd uintptr) {
			bindErr = syscall.SetsockoptString(
				int(fd),
				syscall.SOL_SOCKET,
				syscall.SO_BINDTODEVICE,
				name,
			)
		})
		if err != nil {
			return fmt.Errorf("failed to access socket to bind it to %s: %w", name, err)
		}

		if bindErr != nil {
			return fmt.Errorf("failed to bind socket to interface %s: %w", name, bindErr)
		}

		return nil
	}
}

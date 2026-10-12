// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package speedtest

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

// dialControl is a [net.Dialer] Control function, run on each socket before it connects.
type dialControl = func(network, address string, conn syscall.RawConn) error

// Errors for resolving a network interface as a source.
var (
	// ErrInterfaceNotFound is returned for a name that no network interface has.
	ErrInterfaceNotFound = errors.New("network interface not found")
	// ErrNoInterfaceAddress is returned for an interface without an IPv4 address or an IPv6 address that is not
	// link-local.
	ErrNoInterfaceAddress = errors.New("network interface has no usable address")
)

// sourceBinding is where a client's connections leave from.
type sourceBinding struct {
	// tcp is the local address for TCP dials, or nil.
	tcp net.Addr
	// ip is the local address for ICMP dials, or nil.
	ip net.Addr
	// control binds sockets to the source interface, or nil when the source is not an interface or the platform
	// cannot bind to one.
	control dialControl
	// sourceIP is the address UDP sockets, and DNS queries bound to the source, leave from, or nil.
	sourceIP net.IP
}

// resolveSource resolves a configured source into the addresses and socket control the dialers use.
//
// A source is an IP address, a network interface name, or a hostname, optionally with a network prefix such as
// "tcp://". On Linux, an interface name binds each socket to the device and sets no fixed address, so the kernel
// picks the interface's address in the family of each destination. Other platforms cannot bind to a device and use
// the address [ResolveInterface] picks instead.
//
// Parameters:
//   - source: the configured source, possibly empty.
//
// Returns:
//   - sourceBinding: the binding, empty when source is empty or does not resolve.
func resolveSource(source string) sourceBinding {
	var binding sourceBinding

	if len(source) == 0 {
		return binding
	}

	_, address := parseAddr(source)

	interfaceIP, err := ResolveInterface(address)
	if !errors.Is(err, ErrInterfaceNotFound) {
		binding.control = bindToDevice(address)
		if binding.control != nil {
			return binding
		}

		if err != nil {
			dbg.Printf("Warning: skipping the source interface. err: %s\n", err.Error())

			return binding
		}

		binding.tcp = &net.TCPAddr{IP: interfaceIP}
		binding.ip = &net.IPAddr{IP: interfaceIP}
		binding.sourceIP = interfaceIP

		return binding
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("[%s]:0", address))
	if err == nil {
		binding.tcp = tcpAddr
	} else {
		dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
	}

	ipAddr, err := net.ResolveIPAddr("ip", address)
	if err == nil {
		binding.ip = ipAddr
	} else {
		dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
	}

	binding.sourceIP = net.ParseIP(address)
	if binding.sourceIP == nil && tcpAddr != nil {
		binding.sourceIP = tcpAddr.IP
	}

	return binding
}

// ResolveInterface returns the source address of the network interface with the given name.
//
// It prefers the first IPv4 address and otherwise takes the first IPv6 address that is not link-local, since a
// link-local source cannot reach the internet. Platforms without device binding send from this address.
//
// Parameters:
//   - name: the interface name, such as "eth0".
//
// Returns:
//   - net.IP: the interface's address.
//   - error: an error wrapping [ErrInterfaceNotFound] when no interface has that name, including any IP literal,
//     or wrapping [ErrNoInterfaceAddress] when the interface has no usable address.
func ResolveInterface(name string) (net.IP, error) {
	// An IP literal is never an interface name, and checking first avoids listing interfaces for it.
	if net.ParseIP(name) != nil {
		return nil, fmt.Errorf("%w: %q is an IP address", ErrInterfaceNotFound, name)
	}

	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInterfaceNotFound, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrNoInterfaceAddress, name, err)
	}

	ip := pickInterfaceAddress(addrs)
	if ip == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoInterfaceAddress, name)
	}

	return ip, nil
}

// pickInterfaceAddress picks the source address among an interface's addresses.
//
// Parameters:
//   - addrs: the interface's addresses.
//
// Returns:
//   - net.IP: the first IPv4 address, else the first IPv6 address that is not link-local, else nil.
func pickInterfaceAddress(addrs []net.Addr) net.IP {
	var fallback net.IP

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if ipNet.IP.To4() != nil {
			return ipNet.IP
		}

		if fallback == nil && !ipNet.IP.IsLinkLocalUnicast() {
			fallback = ipNet.IP
		}
	}

	return fallback
}

// chainControl combines dialer Control functions into one that runs each in order and stops at the first error.
//
// Parameters:
//   - controls: the functions to run, where nil entries are skipped.
//
// Returns:
//   - dialControl: the combined function, or nil when every entry is nil.
func chainControl(controls ...dialControl) dialControl {
	var chain []dialControl

	for _, control := range controls {
		if control != nil {
			chain = append(chain, control)
		}
	}

	switch len(chain) {
	case 0:
		return nil
	case 1:
		return chain[0]
	default:
		return func(network, address string, conn syscall.RawConn) error {
			var errs []error

			for _, control := range chain {
				err := control(network, address, conn)
				if err != nil {
					errs = append(errs, err)

					break
				}
			}

			return errors.Join(errs...)
		}
	}
}

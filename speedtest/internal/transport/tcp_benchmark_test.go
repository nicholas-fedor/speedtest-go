// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package transport

import (
	"bufio"
	"net"
	"testing"
)

func Benchmark_pingFormat(b *testing.B) {
	locTime := int64(1234567890)

	for b.Loop() {
		_ = pingFormat(locTime)
	}
}

func BenchmarkClient_ID(b *testing.B) {
	client := &Client{id: "test-id"}

	for b.Loop() {
		_ = client.ID()
	}
}

func BenchmarkClient_Write(b *testing.B) {
	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{conn: client1}

	// Start reader to prevent blocking
	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := client2.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	data := []byte("benchmark test data")

	for b.Loop() {
		_ = client.Write(data)
	}
}

func BenchmarkClient_Read(b *testing.B) {
	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:   client1,
		reader: bufio.NewReader(client1),
	}

	// Start writer
	go func() {
		data := []byte("benchmark response\n")
		for {
			_, err := client2.Write(data)
			if err != nil {
				return
			}
		}
	}()

	for b.Loop() {
		_, _ = client.Read()
	}
}

func BenchmarkClient_Version(b *testing.B) {
	client1, client2 := net.Pipe()

	defer func() { _ = client2.Close() }()

	client := &Client{
		conn:    client1,
		reader:  bufio.NewReader(client1),
		version: "1.0.0", // cached
	}

	for b.Loop() {
		_ = client.Version()
	}
}

// Benchmark memory allocation.
func Benchmark_pingFormat_Alloc(b *testing.B) {
	locTime := int64(1234567890)

	b.ReportAllocs()

	for b.Loop() {
		_ = pingFormat(locTime)
	}
}

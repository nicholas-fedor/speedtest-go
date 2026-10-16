package transport

import (
	"context"
	"net"
	"testing"
)

func Benchmark_generateUUID(b *testing.B) {
	for b.Loop() {
		_, _ = generateUUID()
	}
}

func BenchmarkNewPacketLossSender(b *testing.B) {
	dialer := &net.Dialer{}
	uuid := "test-uuid"

	for b.Loop() {
		_, _ = NewPacketLossSender(uuid, dialer)
	}
}

func BenchmarkPacketLossSender_Send(b *testing.B) {
	// Set up UDP connection for benchmarking
	lc := &net.ListenConfig{}

	listener, err := lc.ListenPacket(context.Background(), "udp", "localhost:0")
	if err != nil {
		b.Skip("Cannot create UDP listener:", err)
	}

	defer func() { _ = listener.Close() }()

	dialer := &net.Dialer{}

	ps, err := NewPacketLossSender("test-uuid", dialer)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	err = ps.Connect(ctx, listener.LocalAddr().String())
	if err != nil {
		b.Skip("Cannot connect:", err)
	}

	// Start reader to consume packets
	go func() {
		buf := make([]byte, 1024)
		for {
			_, _, err := listener.ReadFrom(buf)
			if err != nil {
				return
			}
		}
	}()

	for i := 0; b.Loop(); i++ {
		_ = ps.Send(i % 1000)
	}
}

func BenchmarkPacketLossSender_SendConcurrent(b *testing.B) {
	// Set up UDP connection for benchmarking
	lc := &net.ListenConfig{}

	listener, err := lc.ListenPacket(context.Background(), "udp", "localhost:0")
	if err != nil {
		b.Skip("Cannot create UDP listener:", err)
	}

	defer func() { _ = listener.Close() }()

	dialer := &net.Dialer{}

	ps, err := NewPacketLossSender("test-uuid", dialer)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	err = ps.Connect(ctx, listener.LocalAddr().String())
	if err != nil {
		b.Skip("Cannot connect:", err)
	}

	// Start reader to consume packets
	go func() {
		buf := make([]byte, 1024)
		for {
			_, _, err := listener.ReadFrom(buf)
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = ps.Send(counter % 1000)
			counter++
		}
	})
}

// Benchmark memory allocations.
func Benchmark_generateUUID_Alloc(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, _ = generateUUID()
	}
}

func BenchmarkNewPacketLossSender_Alloc(b *testing.B) {
	dialer := &net.Dialer{}
	uuid := "test-uuid"

	b.ReportAllocs()

	for b.Loop() {
		_, _ = NewPacketLossSender(uuid, dialer)
	}
}

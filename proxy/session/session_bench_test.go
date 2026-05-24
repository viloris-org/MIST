package session

import (
	"io"
	"net"
	"sync/atomic"
	"testing"

	"mist/proxy/padding"
)

func newBenchSession(b *testing.B, enablePadding bool) *Session {
	b.Helper()

	local, remote := net.Pipe()
	b.Cleanup(func() {
		_ = local.Close()
		_ = remote.Close()
	})

	go func() {
		_, _ = io.Copy(io.Discard, remote)
	}()

	s := NewClientSession(local, &padding.DefaultPaddingFactory)
	s.sendPadding = enablePadding
	return s
}

func BenchmarkWriteDataFrame(b *testing.B) {
	payload := make([]byte, 16*1024)
	s := newBenchSession(b, false)

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := s.writeDataFrame(1, payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteDataFrameWithPadding(b *testing.B) {
	payload := make([]byte, 1024)
	s := newBenchSession(b, true)

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := s.writeDataFrame(1, payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteDataFramePaddingWindow(b *testing.B) {
	payload := make([]byte, 1024)
	s := newBenchSession(b, true)
	paddingWrites := int(padding.DefaultPaddingFactory.Load().Stop - 1)

	b.ReportAllocs()
	b.SetBytes(int64(len(payload) * paddingWrites))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.sendPadding = true
		s.pktCounter.Store(0)
		for pkt := 0; pkt < paddingWrites; pkt++ {
			if _, err := s.writeDataFrame(1, payload); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkWriteDataFrameParallel(b *testing.B) {
	payload := make([]byte, 16*1024)
	s := newBenchSession(b, false)
	var sid atomic.Uint32

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		streamID := sid.Add(1)
		for pb.Next() {
			if _, err := s.writeDataFrame(streamID, payload); err != nil {
				b.Fatal(err)
			}
		}
	})
}

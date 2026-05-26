package session

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"mist/proxy/padding"
)

type discardConn struct{}

func (discardConn) Close() error                     { return nil }
func (discardConn) LocalAddr() net.Addr              { return nil }
func (discardConn) RemoteAddr() net.Addr             { return nil }
func (discardConn) SetDeadline(time.Time) error      { return nil }
func (discardConn) SetReadDeadline(time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(time.Time) error { return nil }
func (discardConn) Read([]byte) (int, error)         { return 0, net.ErrClosed }
func (discardConn) Write(b []byte) (int, error)      { return len(b), nil }

func newBenchSession(b *testing.B, enablePadding bool) *Session {
	b.Helper()

	s := NewClientSession(discardConn{}, &padding.DefaultPaddingFactory, 0, 0, 0, 0, 0, nil)
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

func BenchmarkWriteDataFrameHMAC(b *testing.B) {
	payload := make([]byte, 16*1024)
	s := newBenchSession(b, false)
	s.setHMACKey([]byte("0123456789abcdef0123456789abcdef"))
	s.hmacMode = true

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := s.writeDataFrame(1, payload); err != nil {
			b.Fatal(err)
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

package pipe

import (
	"io"
	"testing"
)

func BenchmarkPipeTransfer(b *testing.B) {
	payload := make([]byte, 16*1024)
	reader, writer := Pipe()

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, reader)
		close(done)
	}()

	b.Cleanup(func() {
		_ = writer.Close()
		<-done
	})

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := writer.Write(payload); err != nil {
			b.Fatal(err)
		}
	}
}

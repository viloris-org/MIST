package session

import (
	"bytes"
	"net"
	"testing"
	"time"

	"mist/proxy/padding"
)

type recordingConn struct {
	bytes.Buffer
	writes int
}

func (c *recordingConn) Close() error                     { return nil }
func (c *recordingConn) LocalAddr() net.Addr              { return nil }
func (c *recordingConn) RemoteAddr() net.Addr             { return nil }
func (c *recordingConn) SetDeadline(time.Time) error      { return nil }
func (c *recordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *recordingConn) SetWriteDeadline(time.Time) error { return nil }
func (c *recordingConn) Read([]byte) (int, error)         { return 0, net.ErrClosed }
func (c *recordingConn) Write(b []byte) (int, error) {
	c.writes++
	return c.Buffer.Write(b)
}

func TestWriteDataFrameSplitsLargePayload(t *testing.T) {
	conn := &recordingConn{}
	s := NewClientSession(conn, &padding.DefaultPaddingFactory, 0, 0, 0, 0, 0, nil)
	s.sendPadding = false

	payload := bytes.Repeat([]byte{0x1}, maxFramePayloadLen+10)
	n, err := s.writeDataFrame(7, payload)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("written payload length = %d, want %d", n, len(payload))
	}

	first := rawHeader(conn.Bytes()[:headerOverHeadSize])
	if first.Cmd() != cmdPSH || first.StreamID() != 7 || int(first.Length()) != maxFramePayloadLen {
		t.Fatalf("unexpected first frame header: cmd=%d sid=%d len=%d", first.Cmd(), first.StreamID(), first.Length())
	}

	secondOffset := headerOverHeadSize + maxFramePayloadLen
	second := rawHeader(conn.Bytes()[secondOffset : secondOffset+headerOverHeadSize])
	if second.Cmd() != cmdPSH || second.StreamID() != 7 || int(second.Length()) != 10 {
		t.Fatalf("unexpected second frame header: cmd=%d sid=%d len=%d", second.Cmd(), second.StreamID(), second.Length())
	}
}

func TestWriteDataFrameCapsCoalescedWriteSize(t *testing.T) {
	conn := &recordingConn{}
	s := NewClientSession(conn, &padding.DefaultPaddingFactory, 0, 0, 0, 0, 0, nil)
	s.sendPadding = false

	payload := bytes.Repeat([]byte{0x1}, maxCoalescedWrite+1)
	n, err := s.writeDataFrame(7, payload)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("written payload length = %d, want %d", n, len(payload))
	}
	if conn.writes != 2 {
		t.Fatalf("underlying writes = %d, want 2", conn.writes)
	}
}

func TestWriteControlFrameRejectsOversizedPayload(t *testing.T) {
	conn := &recordingConn{}
	s := NewClientSession(conn, &padding.DefaultPaddingFactory, 0, 0, 0, 0, 0, nil)

	_, err := s.writeControlFrame(frame{cmd: cmdAlert, data: make([]byte, maxFramePayloadLen+1)})
	if err == nil {
		t.Fatal("expected oversized control frame error")
	}
}

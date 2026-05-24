package uot

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"MistCore/common/buf"
	M "MistCore/common/metadata"
)

type discardConn struct {
	bytes.Buffer
}

func (c *discardConn) Close() error                     { return nil }
func (c *discardConn) LocalAddr() net.Addr              { return nil }
func (c *discardConn) RemoteAddr() net.Addr             { return nil }
func (c *discardConn) SetDeadline(time.Time) error      { return nil }
func (c *discardConn) SetReadDeadline(time.Time) error  { return nil }
func (c *discardConn) SetWriteDeadline(time.Time) error { return nil }

type oversizedPacketConn struct{}

func (c *oversizedPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	copy(p, bytes.Repeat([]byte{'a'}, len(p)))
	return math.MaxUint16 + 1, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 53}, nil
}
func (c *oversizedPacketConn) WriteTo([]byte, net.Addr) (int, error) { return 0, nil }
func (c *oversizedPacketConn) Close() error                          { return nil }
func (c *oversizedPacketConn) LocalAddr() net.Addr                   { return &net.UDPAddr{} }
func (c *oversizedPacketConn) SetDeadline(time.Time) error           { return nil }
func (c *oversizedPacketConn) SetReadDeadline(time.Time) error       { return nil }
func (c *oversizedPacketConn) SetWriteDeadline(time.Time) error      { return nil }

func TestConnWriteToRejectsOversizedPayload(t *testing.T) {
	conn := NewConn(&discardConn{}, Request{IsConnect: true, Destination: M.ParseSocksaddr("127.0.0.1:53")})
	payload := bytes.Repeat([]byte{'a'}, math.MaxUint16+1)
	_, err := conn.WriteTo(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 53})
	if err == nil {
		t.Fatal("expected oversized payload error")
	}
}

func TestConnWritePacketRejectsOversizedPayload(t *testing.T) {
	conn := NewConn(&discardConn{}, Request{IsConnect: true, Destination: M.ParseSocksaddr("127.0.0.1:53")})
	buffer := buf.NewSize(math.MaxUint16 + 1)
	defer buffer.Release()
	if _, err := buffer.Write(bytes.Repeat([]byte{'a'}, math.MaxUint16+1)); err != nil {
		t.Fatal(err)
	}
	if err := conn.WritePacket(buffer, M.ParseSocksaddr("127.0.0.1:53")); err == nil {
		t.Fatal("expected oversized payload error")
	}
}

func TestServerConnLoopOutputRejectsOversizedPayload(t *testing.T) {
	serverConn := &ServerConn{PacketConn: &oversizedPacketConn{}}
	serverConn.inputReader, serverConn.inputWriter = io.Pipe()
	serverConn.outputReader, serverConn.outputWriter = io.Pipe()

	errCh := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(serverConn.outputReader)
		errCh <- err
	}()

	serverConn.loopOutput()

	err := <-errCh
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("unexpected read error: %v", err)
	}
}

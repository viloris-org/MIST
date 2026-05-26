package wsconn

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
)

const (
	opContinuation = 0x0
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xa
)

type Conn struct {
	net.Conn
	client bool

	readMu  sync.Mutex
	writeMu sync.Mutex
	readBuf []byte
}

func NewClient(conn net.Conn) net.Conn {
	return &Conn{Conn: conn, client: true}
}

func NewServer(conn net.Conn) net.Conn {
	return &Conn{Conn: conn}
}

func (c *Conn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for len(c.readBuf) == 0 {
		payload, err := c.readFrame()
		if err != nil {
			return 0, err
		}
		c.readBuf = payload
	}
	n := copy(p, c.readBuf)
	c.readBuf = c.readBuf[n:]
	return n, nil
}

func (c *Conn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.writeFrame(opBinary, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *Conn) readFrame() ([]byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(c.Conn, header[:]); err != nil {
		return nil, err
	}
	opcode := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	payloadLen := uint64(header[1] & 0x7f)

	switch payloadLen {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.Conn, ext[:]); err != nil {
			return nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.Conn, ext[:]); err != nil {
			return nil, err
		}
		payloadLen = binary.BigEndian.Uint64(ext[:])
	}
	if payloadLen > uint64(int(^uint(0)>>1)) {
		return nil, errors.New("websocket frame too large")
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.Conn, mask[:]); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, int(payloadLen))
	if _, err := io.ReadFull(c.Conn, payload); err != nil {
		return nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	if c.client && masked {
		return nil, errors.New("server websocket frames must not be masked")
	}
	if !c.client && !masked {
		return nil, errors.New("client websocket frames must be masked")
	}

	switch opcode {
	case opBinary, opContinuation:
		return payload, nil
	case opClose:
		return nil, io.EOF
	case opPing:
		c.writeMu.Lock()
		err := c.writeFrame(opPong, payload)
		c.writeMu.Unlock()
		if err != nil {
			return nil, err
		}
		return c.readFrame()
	case opPong:
		return c.readFrame()
	default:
		return nil, errors.New("unsupported websocket opcode")
	}
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	var header [14]byte
	header[0] = 0x80 | opcode
	off := 2
	mask := c.client
	if len(payload) < 126 {
		header[1] = byte(len(payload))
	} else if len(payload) <= 0xffff {
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:4], uint16(len(payload)))
		off = 4
	} else {
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:10], uint64(len(payload)))
		off = 10
	}

	if mask {
		header[1] |= 0x80
		if _, err := io.ReadFull(rand.Reader, header[off:off+4]); err != nil {
			return err
		}
		off += 4
	}
	if err := writeAll(c.Conn, header[:off]); err != nil {
		return err
	}
	if !mask {
		return writeAll(c.Conn, payload)
	}

	maskKey := header[off-4 : off]
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ maskKey[i%4]
	}
	return writeAll(c.Conn, masked)
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}

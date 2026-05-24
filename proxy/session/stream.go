package session

import (
	"io"
	"net"
	"os"
	"sync"
	"time"

	"MistCore/common/buf"
)

const streamReadBuffer = 16

// Stream implements net.Conn
type Stream struct {
	id uint32

	sess *Session

	dataCh  chan []byte
	dataBuf []byte
	dataOff int
	readMu  sync.Mutex
	die     chan struct{}

	writeDeadline pipeDeadline

	dieOnce sync.Once
	dieHook func()
	dieErr  error

	reportOnce sync.Once
}

// newStream initiates a Stream struct
func newStream(id uint32, sess *Session) *Stream {
	s := new(Stream)
	s.id = id
	s.sess = sess
	s.dataCh = make(chan []byte, streamReadBuffer)
	s.die = make(chan struct{})
	return s
}

// Read implements net.Conn
func (s *Stream) Read(b []byte) (n int, err error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	// consume remaining data from previous frame
	if s.dataOff < len(s.dataBuf) {
		n = copy(b, s.dataBuf[s.dataOff:])
		s.dataOff += n
		if s.dataOff >= len(s.dataBuf) {
			buf.Put(s.dataBuf)
			s.dataBuf = nil
			s.dataOff = 0
		}
		return n, nil
	}

	// get next frame
	select {
	case frame, ok := <-s.dataCh:
		if !ok {
			if s.dieErr != nil {
				return 0, s.dieErr
			}
			return 0, io.EOF
		}
		s.dataBuf = frame
		n = copy(b, frame)
		s.dataOff = n
		if n >= len(frame) {
			buf.Put(frame)
			s.dataBuf = nil
			s.dataOff = 0
		}
		return n, nil
	case <-s.die:
		if s.dieErr != nil {
			return 0, s.dieErr
		}
		return 0, io.ErrClosedPipe
	}
}

// Write implements net.Conn
func (s *Stream) Write(b []byte) (n int, err error) {
	select {
	case <-s.writeDeadline.Wait():
		return 0, os.ErrDeadlineExceeded
	default:
	}
	if s.dieErr != nil {
		return 0, s.dieErr
	}
	n, err = s.sess.writeDataFrame(s.id, b)
	return
}

// Close implements net.Conn
func (s *Stream) Close() error {
	return s.closeWithError(io.ErrClosedPipe)
}

// closeLocally only closes Stream and don't notify remote peer
func (s *Stream) closeLocally() {
	var once bool
	s.dieOnce.Do(func() {
		s.dieErr = net.ErrClosed
		close(s.die)
		s.drainDataCh()
		once = true
	})
	if once {
		if s.dieHook != nil {
			s.dieHook()
			s.dieHook = nil
		}
	}
}

func (s *Stream) closeWithError(err error) error {
	var once bool
	s.dieOnce.Do(func() {
		s.dieErr = err
		close(s.die)
		s.drainDataCh()
		once = true
	})
	if once {
		if s.dieHook != nil {
			s.dieHook()
			s.dieHook = nil
		}
		return s.sess.streamClosed(s.id)
	} else {
		return s.dieErr
	}
}

func (s *Stream) drainDataCh() {
	// drain and release any pending buffers
	s.readMu.Lock()
	if s.dataBuf != nil && s.dataOff < len(s.dataBuf) {
		buf.Put(s.dataBuf)
		s.dataBuf = nil
		s.dataOff = 0
	}
	s.readMu.Unlock()

	for {
		select {
		case frame := <-s.dataCh:
			buf.Put(frame)
		default:
			return
		}
	}
}

func (s *Stream) deliverData(frame []byte) {
	select {
	case s.dataCh <- frame:
	case <-s.die:
		buf.Put(frame)
	}
}

func (s *Stream) SetReadDeadline(t time.Time) error {
	return nil
}

func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.writeDeadline.Set(t)
	return nil
}

func (s *Stream) SetDeadline(t time.Time) error {
	s.SetWriteDeadline(t)
	return s.SetReadDeadline(t)
}

// LocalAddr satisfies net.Conn interface
func (s *Stream) LocalAddr() net.Addr {
	if ts, ok := s.sess.conn.(interface {
		LocalAddr() net.Addr
	}); ok {
		return ts.LocalAddr()
	}
	return nil
}

// RemoteAddr satisfies net.Conn interface
func (s *Stream) RemoteAddr() net.Addr {
	if ts, ok := s.sess.conn.(interface {
		RemoteAddr() net.Addr
	}); ok {
		return ts.RemoteAddr()
	}
	return nil
}

// HandshakeFailure should be called when Server fail to create outbound proxy
func (s *Stream) HandshakeFailure(err error) error {
	var once bool
	s.reportOnce.Do(func() {
		once = true
	})
	if once && err != nil && s.sess.peerVersion >= 2 {
		f := newFrame(cmdSYNACK, s.id)
		f.data = []byte(err.Error())
		if _, err := s.sess.writeControlFrame(f); err != nil {
			return err
		}
	}
	return nil
}

// HandshakeSuccess should be called when Server success to create outbound proxy
func (s *Stream) HandshakeSuccess() error {
	var once bool
	s.reportOnce.Do(func() {
		once = true
	})
	if once && s.sess.peerVersion >= 2 {
		if _, err := s.sess.writeControlFrame(newFrame(cmdSYNACK, s.id)); err != nil {
			return err
		}
	}
	return nil
}

type pipeDeadline struct {
	mu     sync.Mutex
	timer  *time.Timer
	cancel chan struct{}
}

func (d *pipeDeadline) Set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	if d.cancel == nil {
		d.cancel = make(chan struct{})
	}
	if t.IsZero() {
		return
	}
	select {
	case <-d.cancel:
		d.cancel = make(chan struct{})
	default:
	}
	d.timer = time.AfterFunc(time.Until(t), func() {
		close(d.cancel)
	})
}

func (d *pipeDeadline) Wait() chan struct{} {
	d.mu.Lock()
	if d.cancel == nil {
		d.cancel = make(chan struct{})
	}
	ch := d.cancel
	d.mu.Unlock()
	return ch
}

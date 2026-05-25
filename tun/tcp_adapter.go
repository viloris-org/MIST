//go:build linux

package tun

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/netip"
	"sync"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/waiter"
)

var errTimeout = errors.New("i/o timeout")

// tcpAddr implements net.Addr for a netip.AddrPort.
type tcpAddr struct {
	ap netip.AddrPort
}

func (a tcpAddr) Network() string { return "tcp" }
func (a tcpAddr) String() string  { return a.ap.String() }

// deadlineController manages read/write deadlines for a net.Conn.
type deadlineController struct {
	mu           sync.Mutex
	readDeadline time.Time
	readTimer    *time.Timer
}

func (d *deadlineController) SetReadDeadline(t time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.readDeadline = t
	return nil
}

func (d *deadlineController) SetWriteDeadline(t time.Time) error {
	return nil // writes are non-blocking
}

func (d *deadlineController) SetDeadline(t time.Time) error {
	d.SetReadDeadline(t)
	d.SetWriteDeadline(t)
	return nil
}

type tcpConn struct {
	ep tcpip.Endpoint
	wq *waiter.Queue
	id stack.TransportEndpointID

	deadline deadlineController

	readMu sync.Mutex

	local  net.Addr
	remote net.Addr
}

func newTCPConn(wq *waiter.Queue, ep tcpip.Endpoint, id stack.TransportEndpointID) *tcpConn {
	localIP, _ := netip.AddrFromSlice(id.LocalAddress.AsSlice())
	remoteIP, _ := netip.AddrFromSlice(id.RemoteAddress.AsSlice())

	localAddr := tcpAddr{netip.AddrPortFrom(localIP.Unmap(), id.LocalPort)}
	remoteAddr := tcpAddr{netip.AddrPortFrom(remoteIP.Unmap(), id.RemotePort)}

	return &tcpConn{
		ep:     ep,
		wq:     wq,
		id:     id,
		local:  localAddr,
		remote: remoteAddr,
	}
}

func (c *tcpConn) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	w := tcpip.SliceWriter(b)
	opts := tcpip.ReadOptions{}
	res, err := c.ep.Read(&w, opts)

	if _, ok := err.(*tcpip.ErrWouldBlock); ok {
		waitEntry, notifyCh := waiter.NewChannelEntry(waiter.ReadableEvents)
		c.wq.EventRegister(&waitEntry)
		defer c.wq.EventUnregister(&waitEntry)
		for {
			res, err = c.ep.Read(&w, opts)
			if _, ok := err.(*tcpip.ErrWouldBlock); !ok {
				break
			}
			select {
			case <-notifyCh:
			}
		}
	}

	if err != nil {
		return 0, wrapError(err)
	}
	return res.Count, nil
}

func (c *tcpConn) Write(b []byte) (int, error) {
	opts := tcpip.WriteOptions{}
	n, err := c.ep.Write(bytes.NewReader(b), opts)

	if _, ok := err.(*tcpip.ErrWouldBlock); ok {
		waitEntry, notifyCh := waiter.NewChannelEntry(waiter.WritableEvents)
		c.wq.EventRegister(&waitEntry)
		defer c.wq.EventUnregister(&waitEntry)
		for {
			n, err = c.ep.Write(bytes.NewReader(b), opts)
			if _, ok := err.(*tcpip.ErrWouldBlock); !ok {
				break
			}
			select {
			case <-notifyCh:
			}
		}
	}

	if err != nil {
		return 0, wrapError(err)
	}
	return int(n), nil
}

func (c *tcpConn) Close() error {
	c.ep.Close()
	return nil
}

func (c *tcpConn) LocalAddr() net.Addr  { return c.local }
func (c *tcpConn) RemoteAddr() net.Addr { return c.remote }

func (c *tcpConn) SetDeadline(t time.Time) error      { return c.deadline.SetDeadline(t) }
func (c *tcpConn) SetReadDeadline(t time.Time) error  { return c.deadline.SetReadDeadline(t) }
func (c *tcpConn) SetWriteDeadline(t time.Time) error { return c.deadline.SetWriteDeadline(t) }

var _ net.Conn = (*tcpConn)(nil)

func wrapError(err tcpip.Error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*tcpip.ErrClosedForReceive); ok {
		return io.EOF
	}
	if _, ok := err.(*tcpip.ErrClosedForSend); ok {
		return io.ErrClosedPipe
	}
	if _, ok := err.(*tcpip.ErrConnectionReset); ok {
		return io.ErrClosedPipe
	}
	return errors.New(err.String())
}

package tun

import (
	"bytes"
	"errors"
	"io"
	"net/netip"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/waiter"
)

type udpConn struct {
	ep     tcpip.Endpoint
	wq     *waiter.Queue
	id     stack.TransportEndpointID
	remote netip.AddrPort
}

func newUDPConn(wq *waiter.Queue, ep tcpip.Endpoint, id stack.TransportEndpointID) *udpConn {
	remoteIP, _ := netip.AddrFromSlice(id.LocalAddress.AsSlice())
	return &udpConn{
		ep:     ep,
		wq:     wq,
		id:     id,
		remote: netip.AddrPortFrom(remoteIP.Unmap(), id.LocalPort),
	}
}

func (c *udpConn) ReadPacket() ([]byte, netip.AddrPort, error) {
	buf := make([]byte, 65535)
	w := tcpip.SliceWriter(buf)
	opts := tcpip.ReadOptions{NeedRemoteAddr: true}
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
		return nil, netip.AddrPort{}, wrapError(err)
	}

	n := res.Count
	data := make([]byte, n)
	copy(data, buf[:n])

	remote := c.remote
	if res.RemoteAddr.Addr.Len() > 0 {
		if ip, ok := netip.AddrFromSlice(res.RemoteAddr.Addr.AsSlice()); ok {
			remote = netip.AddrPortFrom(ip.Unmap(), res.RemoteAddr.Port)
		}
	}

	return data, remote, nil
}

func (c *udpConn) WritePacket(data []byte, _ netip.AddrPort) error {
	opts := tcpip.WriteOptions{}
	n, err := c.ep.Write(bytes.NewReader(data), opts)

	if _, ok := err.(*tcpip.ErrWouldBlock); ok {
		waitEntry, notifyCh := waiter.NewChannelEntry(waiter.WritableEvents)
		c.wq.EventRegister(&waitEntry)
		defer c.wq.EventUnregister(&waitEntry)
		for {
			n, err = c.ep.Write(bytes.NewReader(data), opts)
			if _, ok := err.(*tcpip.ErrWouldBlock); !ok {
				break
			}
			select {
			case <-notifyCh:
			}
		}
	}

	if err != nil {
		return wrapError(err)
	}
	_ = n
	return nil
}

func (c *udpConn) Close() error {
	c.ep.Close()
	return nil
}

func (c *udpConn) RemoteAddr() netip.AddrPort {
	return c.remote
}

var _ UDPConn = (*udpConn)(nil)

// Avoid unused imports
var _ = io.EOF
var _ = errors.New

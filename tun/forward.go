//go:build linux

package tun

import (
	"context"
	"net/netip"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const (
	defaultTCPReceiveBuffer  = 0 // use system default
	maxInFlightTCPConnections = 1024
)

func registerForwarders(s *gvisorStack, handler Handler) error {
	registerTCPForwarder(s, handler)
	registerUDPForwarder(s, handler)
	return nil
}

func registerTCPForwarder(s *gvisorStack, handler Handler) {
	fwd := tcp.NewForwarder(s.Stack(), defaultTCPReceiveBuffer, maxInFlightTCPConnections, func(req *tcp.ForwarderRequest) {
		go handleTCPForwardRequest(req, handler)
	})
	s.Stack().SetTransportProtocolHandler(tcp.ProtocolNumber, fwd.HandlePacket)
}

func handleTCPForwardRequest(req *tcp.ForwarderRequest, handler Handler) {
	id := req.ID()

	dest := extractAddrPort(id.LocalAddress, id.LocalPort)
	if !dest.IsValid() {
		req.Complete(true) // send RST
		return
	}

	wq := &waiter.Queue{}
	ep, err := req.CreateEndpoint(wq)
	if err != nil {
		req.Complete(true)
		return
	}

	conn := newTCPConn(wq, ep, id)
	defer conn.Close()

	ctx := context.Background()
	if hErr := handler.HandleTCP(ctx, conn, dest); hErr != nil {
		ep.Close()
	}
}

func registerUDPForwarder(s *gvisorStack, handler Handler) {
	fwd := udp.NewForwarder(s.Stack(), func(req *udp.ForwarderRequest) {
		go handleUDPForwardRequest(req, handler)
	})
	s.Stack().SetTransportProtocolHandler(udp.ProtocolNumber, fwd.HandlePacket)
}

func handleUDPForwardRequest(req *udp.ForwarderRequest, handler Handler) {
	id := req.ID()

	dest := extractAddrPort(id.LocalAddress, id.LocalPort)
	if !dest.IsValid() {
		return
	}

	wq := &waiter.Queue{}
	ep, err := req.CreateEndpoint(wq)
	if err != nil {
		return
	}

	conn := newUDPConn(wq, ep, id)
	defer conn.Close()

	ctx := context.Background()
	handler.HandleUDP(ctx, conn, dest)
}

func extractAddrPort(addr tcpip.Address, port uint16) netip.AddrPort {
	ip, ok := netip.AddrFromSlice(addr.AsSlice())
	if !ok {
		return netip.AddrPort{}
	}
	ip = ip.Unmap()
	return netip.AddrPortFrom(ip, port)
}

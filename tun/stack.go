//go:build linux

package tun

import (
	"fmt"

	"golang.zx2c4.com/wireguard/tun"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

const nicID = tcpip.NICID(1)

type gvisorStack struct {
	s      *stack.Stack
	linkEP stack.LinkEndpoint
}

func createStack(tunDev tun.Device, mtu int) (*gvisorStack, error) {
	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
		HandleLocal: true,
	})

	// Wrap the TUN file descriptor as a fdbased link endpoint.
	tunFile := tunDev.File()
	if tunFile == nil {
		return nil, fmt.Errorf("TUN device has no backing file")
	}
	fd := int(tunFile.Fd())

	linkEP, err := fdbased.New(&fdbased.Options{
		FDs:  []int{fd},
		MTU:  uint32(mtu),
	})
	if err != nil {
		return nil, fmt.Errorf("create fdbased endpoint: %w", err)
	}

	if err := s.CreateNIC(nicID, linkEP); err != nil {
		return nil, fmt.Errorf("create NIC: %s", err)
	}

	return &gvisorStack{
		s:      s,
		linkEP: linkEP,
	}, nil
}

func (gs *gvisorStack) Start() error {
	// Enable forwarding on the stack.
	gs.s.SetForwardingDefaultAndAllNICs(ipv4.ProtocolNumber, true)
	gs.s.SetForwardingDefaultAndAllNICs(ipv6.ProtocolNumber, true)

	return nil
}

func (gs *gvisorStack) Close() {
	gs.s.RemoveNIC(nicID)
	gs.s.Close()
}

// Stack returns the underlying gVisor stack.
func (gs *gvisorStack) Stack() *stack.Stack {
	return gs.s
}

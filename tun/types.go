package tun

import (
	"context"
	"net"
	"net/netip"
)

// Config holds TUN device configuration.
type Config struct {
	Name    string
	MTU     int
	Address netip.Prefix
	DNS     []netip.Addr
	Routes  []netip.Prefix
}

// Stats holds runtime TUN device metrics.
type Stats struct {
	Name      string `json:"name"`
	MTU       int    `json:"mtu"`
	Address   string `json:"address"`
	IsUp      bool   `json:"is_up"`
	RxBytes   int64  `json:"rx_bytes"`
	TxBytes   int64  `json:"tx_bytes"`
	RxPackets int64  `json:"rx_packets"`
	TxPackets int64  `json:"tx_packets"`
}

// Handler receives connections and packets intercepted from the TUN interface.
type Handler interface {
	HandleTCP(ctx context.Context, conn net.Conn, dest netip.AddrPort) error
	HandleUDP(ctx context.Context, conn UDPConn, dest netip.AddrPort) error
}

// UDPConn is a packet-oriented interface for UDP datagrams from the TUN.
type UDPConn interface {
	ReadPacket() ([]byte, netip.AddrPort, error)
	WritePacket(data []byte, dest netip.AddrPort) error
	Close() error
}

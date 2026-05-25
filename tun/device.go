package tun

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"

	netlinkTun "golang.zx2c4.com/wireguard/tun"
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

// Device manages a TUN interface and its associated gVisor network stack.
type Device struct {
	cfg     Config
	name    string
	mtu     int
	handler Handler

	tunDev netlinkTun.Device
	stack  *gvisorStack

	die    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Create creates a new TUN device and initializes the gVisor network stack.
func Create(cfg Config, handler Handler) (*Device, error) {
	if cfg.MTU == 0 {
		cfg.MTU = 1500
	}
	if cfg.Name == "" {
		cfg.Name = "mist"
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Device{
		cfg:     cfg,
		mtu:     cfg.MTU,
		handler: handler,
		die:     ctx,
		cancel:  cancel,
	}

	tunDev, err := netlinkTun.CreateTUN(cfg.Name, cfg.MTU)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create tun device %q: %w", cfg.Name, err)
	}
	d.tunDev = tunDev

	name, err := tunDev.Name()
	if err != nil {
		name = cfg.Name
	}
	d.name = name

	stk, err := createStack(tunDev, cfg.MTU)
	if err != nil {
		tunDev.Close()
		cancel()
		return nil, fmt.Errorf("create stack: %w", err)
	}
	d.stack = stk

	if err := registerForwarders(stk, handler); err != nil {
		tunDev.Close()
		cancel()
		return nil, fmt.Errorf("register forwarders: %w", err)
	}

	return d, nil
}

// Start activates the TUN device — configures the IP address, brings the link
// up, and begins forwarding traffic.
func (d *Device) Start() error {
	if err := d.stack.Start(); err != nil {
		return fmt.Errorf("start stack: %w", err)
	}

	if err := configureInterface(d.name, d.cfg); err != nil {
		return fmt.Errorf("configure interface: %w", err)
	}

	return nil
}

// Close shuts down the TUN device and releases all resources.
func (d *Device) Close() error {
	d.cancel()
	d.stack.Close()
	return d.tunDev.Close()
}

// Name returns the actual OS interface name.
func (d *Device) Name() string {
	return d.name
}

// MTU returns the configured MTU.
func (d *Device) MTU() int {
	return d.mtu
}

// Stats returns current device statistics.
func (d *Device) Stats() Stats {
	return Stats{
		Name:    d.name,
		MTU:     d.mtu,
		Address: d.cfg.Address.String(),
		IsUp:    true,
	}
}

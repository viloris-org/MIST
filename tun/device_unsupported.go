//go:build !linux

package tun

import "fmt"

// Device is a stub for non-Linux platforms.
type Device struct {
	cfg Config
}

// Create is a stub that always returns an error on unsupported platforms.
func Create(cfg Config, handler Handler) (*Device, error) {
	return nil, fmt.Errorf("TUN device not supported on this platform")
}

// Start is a stub.
func (d *Device) Start() error {
	return fmt.Errorf("TUN device not supported on this platform")
}

// Close is a stub.
func (d *Device) Close() error {
	return nil
}

// Name is a stub.
func (d *Device) Name() string {
	return ""
}

// MTU is a stub.
func (d *Device) MTU() int {
	return 0
}

// Stats is a stub.
func (d *Device) Stats() Stats {
	return Stats{}
}

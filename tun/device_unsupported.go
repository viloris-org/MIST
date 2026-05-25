//go:build !linux

package tun

import "fmt"

func configureInterface(name string, cfg Config) error {
	return fmt.Errorf("TUN device configuration not supported on this platform")
}

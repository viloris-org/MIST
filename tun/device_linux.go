package tun

import (
	"fmt"
	"os/exec"
)

func configureInterface(name string, cfg Config) error {
	// Assign IP address.
	addr := cfg.Address.String()
	if err := exec.Command("ip", "addr", "add", addr, "dev", name).Run(); err != nil {
		return fmt.Errorf("ip addr add %s dev %s: %w", addr, name, err)
	}

	// Bring the interface up.
	if err := exec.Command("ip", "link", "set", name, "up").Run(); err != nil {
		return fmt.Errorf("ip link set %s up: %w", name, err)
	}

	// Add routes.
	for _, route := range cfg.Routes {
		out, err := exec.Command("ip", "route", "add", route.String(), "dev", name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("ip route add %s dev %s: %w (%s)", route, name, err, out)
		}
	}

	return nil
}

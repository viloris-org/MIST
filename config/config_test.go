package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.toml")
	content := `
server = "example.com:8443"
password = "secret"

[tls]
sni = "example.com"
min_version = "1.3"
min_idle_session = 10
tls_profile = "web"
traffic_profile = "api"

[tun]
enabled = true
address = "10.1.0.2/24"
mtu = 1400

[dns]
enabled = true
listen = "127.0.0.1:5354"

`
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := DecodeFile(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server != "example.com:8443" {
		t.Errorf("Server = %q, want example.com:8443", cfg.Server)
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %q, want secret", cfg.Password)
	}
	if cfg.TLS.SNI != "example.com" {
		t.Errorf("TLS.SNI = %q, want example.com", cfg.TLS.SNI)
	}
	if cfg.TLS.MinVersion != "1.3" {
		t.Errorf("TLS.MinVersion = %q, want 1.3", cfg.TLS.MinVersion)
	}
	if cfg.TLS.MinIdleSession != 10 {
		t.Errorf("TLS.MinIdleSession = %d, want 10", cfg.TLS.MinIdleSession)
	}
	if cfg.TLS.TLSProfile != "web" {
		t.Errorf("TLS.TLSProfile = %q, want web", cfg.TLS.TLSProfile)
	}
	if cfg.TLS.TrafficProfile != "api" {
		t.Errorf("TLS.TrafficProfile = %q, want api", cfg.TLS.TrafficProfile)
	}
	if !cfg.Tun.Enabled {
		t.Error("Tun.Enabled = false, want true")
	}
	if cfg.Tun.Address != "10.1.0.2/24" {
		t.Errorf("Tun.Address = %q", cfg.Tun.Address)
	}
	if cfg.Tun.MTU != 1400 {
		t.Errorf("Tun.MTU = %d, want 1400", cfg.Tun.MTU)
	}
	if !cfg.DNS.Enabled {
		t.Error("DNS.Enabled = false, want true")
	}
	if cfg.DNS.Listen != "127.0.0.1:5354" {
		t.Errorf("DNS.Listen = %q", cfg.DNS.Listen)
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &ClientConfig{}
	cfg.SetDefaults()

	if cfg.TLS.MinVersion != "1.2" {
		t.Errorf("TLS.MinVersion = %q, want 1.2", cfg.TLS.MinVersion)
	}
	if cfg.TLS.MinIdleSession != 5 {
		t.Errorf("TLS.MinIdleSession = %d, want 5", cfg.TLS.MinIdleSession)
	}
	if cfg.TLS.TLSProfile != "default" {
		t.Errorf("TLS.TLSProfile = %q, want default", cfg.TLS.TLSProfile)
	}
	if cfg.TLS.TrafficProfile != "web" {
		t.Errorf("TLS.TrafficProfile = %q, want web", cfg.TLS.TrafficProfile)
	}
	if cfg.Inbound.Listen != "127.0.0.1:1080" {
		t.Errorf("Inbound.Listen = %q", cfg.Inbound.Listen)
	}
	if len(cfg.Inbound.Types) != 2 || cfg.Inbound.Types[0] != "socks" || cfg.Inbound.Types[1] != "http" {
		t.Errorf("Inbound.Types = %v", cfg.Inbound.Types)
	}
	if cfg.Tun.Name != "mist" {
		t.Errorf("Tun.Name = %q, want mist", cfg.Tun.Name)
	}
	if cfg.Tun.MTU != 1500 {
		t.Errorf("Tun.MTU = %d, want 1500", cfg.Tun.MTU)
	}
	if cfg.Tun.Address != "10.0.0.2/24" {
		t.Errorf("Tun.Address = %q", cfg.Tun.Address)
	}
	if cfg.DNS.Listen != "127.0.0.1:5353" {
		t.Errorf("DNS.Listen = %q", cfg.DNS.Listen)
	}
}

func TestApplyCLIOverrides(t *testing.T) {
	cfg := &ClientConfig{
		Server:   "config.example.com:8443",
		Password: "config-pass",
		TLS: TLSConfig{
			SNI:        "config.example.com",
			MinVersion: "1.2",
		},
		Tun: TunConfig{
			Name:    "cfg-tun",
			Enabled: false,
		},
	}

	cli := &CLIOverrides{
		Server:        "cli.example.com:8443",
		TLSMinVersion: "1.3",
		Tun:           true,
	}

	cfg.ApplyCLIOverrides(cli)

	if cfg.Server != "cli.example.com:8443" {
		t.Errorf("Server = %q, CLI should override", cfg.Server)
	}
	if cfg.Password != "config-pass" {
		t.Errorf("Password = %q, should keep config value", cfg.Password)
	}
	if cfg.TLS.MinVersion != "1.3" {
		t.Errorf("TLS.MinVersion = %q, should be CLI value 1.3", cfg.TLS.MinVersion)
	}
	if !cfg.Tun.Enabled {
		t.Error("Tun.Enabled should be true from CLI")
	}
	if cfg.Tun.Name != "cfg-tun" {
		t.Errorf("Tun.Name = %q, should keep config value", cfg.Tun.Name)
	}
}

func TestDecodeFileNotFound(t *testing.T) {
	_, err := DecodeFile("/nonexistent/path.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDecodeFileInvalid(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.toml")
	os.WriteFile(tmp, []byte("[[[]]]broken == toml"), 0644)
	_, err := DecodeFile(tmp)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

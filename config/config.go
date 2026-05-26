package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ClientConfig is the TOML-based client configuration.
type ClientConfig struct {
	Server   string `toml:"server"`
	Password string `toml:"password"`

	TLS     TLSConfig     `toml:"tls"`
	Inbound InboundConfig `toml:"inbound"`
	Tun     TunConfig     `toml:"tun"`
	DNS     DNSConfig     `toml:"dns"`
	Log     LogConfig     `toml:"log"`
}

type TLSConfig struct {
	SNI            string `toml:"sni"`
	CertSHA256     string `toml:"cert_sha256"`
	Insecure       bool   `toml:"insecure"`
	MinVersion     string `toml:"min_version"`
	MinIdleSession int    `toml:"min_idle_session"`
	Transport      string `toml:"transport"`
	TLSProfile     string `toml:"tls_profile"`
	TrafficProfile string `toml:"traffic_profile"`
}

type InboundConfig struct {
	Listen         string   `toml:"listen"`
	Types          []string `toml:"types"`
	RedirectListen string   `toml:"redirect_listen"`
}

type TunConfig struct {
	Enabled bool     `toml:"enabled"`
	Name    string   `toml:"name"`
	MTU     int      `toml:"mtu"`
	Address string   `toml:"address"`
	DNS     []string `toml:"dns"`
	Routes  []string `toml:"routes"`
}

type DNSConfig struct {
	Enabled  bool     `toml:"enabled"`
	Listen   string   `toml:"listen"`
	Upstream []string `toml:"upstream"`
}

type LogConfig struct {
	Format string `toml:"format"`
	File   string `toml:"file"`
	Level  string `toml:"level"`
}

// DecodeFile reads and decodes a TOML config file.
func DecodeFile(path string) (*ClientConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	cfg := &ClientConfig{}
	if _, err := toml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// SetDefaults fills in default values for a ClientConfig.
func (c *ClientConfig) SetDefaults() {
	if c.TLS.MinVersion == "" {
		c.TLS.MinVersion = "1.2"
	}
	if c.TLS.MinIdleSession == 0 {
		c.TLS.MinIdleSession = 5
	}
	if c.TLS.Transport == "" {
		c.TLS.Transport = "tls"
	}
	if c.TLS.TLSProfile == "" {
		c.TLS.TLSProfile = "default"
	}
	if c.TLS.TrafficProfile == "" {
		c.TLS.TrafficProfile = "api"
	}
	if c.Inbound.Listen == "" {
		c.Inbound.Listen = "127.0.0.1:1080"
	}
	if len(c.Inbound.Types) == 0 {
		c.Inbound.Types = []string{"socks", "http"}
	}
	if c.Inbound.RedirectListen == "" {
		c.Inbound.RedirectListen = "127.0.0.1:12345"
	}
	if c.Tun.Name == "" {
		c.Tun.Name = "mist"
	}
	if c.Tun.MTU == 0 {
		c.Tun.MTU = 1500
	}
	if c.Tun.Address == "" {
		c.Tun.Address = "10.0.0.2/24"
	}
	if c.Tun.DNS == nil {
		c.Tun.DNS = []string{"1.1.1.1", "8.8.8.8"}
	}
	if c.DNS.Listen == "" {
		c.DNS.Listen = "127.0.0.1:5353"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}
}

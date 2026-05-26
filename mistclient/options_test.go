package mistclient

import (
	"crypto/tls"
	"testing"
)

func TestTLSConfigSNIWithExplicitSNI(t *testing.T) {
	opts := Options{
		ServerAddr: "203.0.113.10:8443",
		SNI:        "example.com",
	}
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "example.com" {
		t.Fatalf("ServerName = %q, want example.com", cfg.ServerName)
	}
}

func TestTLSConfigSNIUsesDomainHost(t *testing.T) {
	opts := Options{
		ServerAddr: "example.com:8443",
	}
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "example.com" {
		t.Fatalf("ServerName = %q, want example.com", cfg.ServerName)
	}
}

func TestTLSConfigSNIHidesSNIForIPHost(t *testing.T) {
	opts := Options{
		ServerAddr: "203.0.113.10:8443",
	}
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "" {
		t.Fatalf("ServerName = %q, want empty (no SNI for IP)", cfg.ServerName)
	}
}

func TestTLSConfigCertPinVerification(t *testing.T) {
	opts := Options{
		ServerAddr:    "example.com:8443",
		TLSCertSHA256: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	}
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should be true when cert pin is set")
	}
	if cfg.VerifyPeerCertificate == nil {
		t.Fatal("VerifyPeerCertificate should be set when cert pin is set")
	}
}

func TestOptionsValidateMissingServerAddr(t *testing.T) {
	opts := Options{Password: "test"}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error for missing ServerAddr")
	}
}

func TestOptionsValidateMissingPassword(t *testing.T) {
	opts := Options{ServerAddr: "example.com:8443"}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected error for missing Password")
	}
}

func TestOptionsValidateRejectsInvalidCertPin(t *testing.T) {
	opts := Options{
		ServerAddr:    "example.com:8443",
		Password:      "test",
		TLSCertSHA256: "not-hex",
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected invalid TLSCertSHA256 to fail validation")
	}
}

func TestOptionsSetDefaults(t *testing.T) {
	opts := Options{ServerAddr: "example.com:8443", Password: "test"}
	opts.SetDefaults()
	if opts.TLSMinVersion != tls.VersionTLS12 {
		t.Fatal("default TLSMinVersion should be 1.2")
	}
	if opts.MinIdleSession != 1 {
		t.Fatal("default MinIdleSession should be 1")
	}
	if opts.Logger == nil {
		t.Fatal("default Logger should not be nil")
	}
	if opts.Transport != "tls" {
		t.Fatalf("default Transport = %q, want tls", opts.Transport)
	}
	if opts.TLSProfile != "default" {
		t.Fatalf("default TLSProfile = %q, want default", opts.TLSProfile)
	}
	if opts.TrafficProfile != "api" {
		t.Fatalf("default TrafficProfile = %q, want api", opts.TrafficProfile)
	}
}

func TestTLSConfigDefaultTransportDoesNotAdvertiseHTTP(t *testing.T) {
	opts := Options{ServerAddr: "example.com:8443", Password: "test"}
	opts.SetDefaults()
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NextProtos) != 0 {
		t.Fatalf("NextProtos = %v, want empty for tls transport", cfg.NextProtos)
	}
}

func TestTLSConfigWebProfileAdvertisesHTTP2AndHTTP11(t *testing.T) {
	opts := Options{ServerAddr: "example.com:8443", Password: "test", TLSProfile: "web"}
	opts.SetDefaults()
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NextProtos) != 2 || cfg.NextProtos[0] != "h2" || cfg.NextProtos[1] != "http/1.1" {
		t.Fatalf("NextProtos = %v, want [h2 http/1.1]", cfg.NextProtos)
	}
}

func TestOptionsValidateRejectsInvalidTrafficProfile(t *testing.T) {
	opts := Options{
		ServerAddr:     "example.com:8443",
		Password:       "test",
		TrafficProfile: "unknown",
	}
	if err := opts.Validate(); err == nil {
		t.Fatal("expected invalid TrafficProfile to fail validation")
	}
}

func TestTLSConfigWSSTransportAdvertisesHTTP11(t *testing.T) {
	opts := Options{ServerAddr: "example.com:8443", Password: "test", Transport: "wss"}
	opts.SetDefaults()
	cfg, err := opts.TLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.NextProtos) != 1 || cfg.NextProtos[0] != "http/1.1" {
		t.Fatalf("NextProtos = %v, want [http/1.1]", cfg.NextProtos)
	}
}

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
	cfg := opts.TLSConfig()
	if cfg.ServerName != "example.com" {
		t.Fatalf("ServerName = %q, want example.com", cfg.ServerName)
	}
}

func TestTLSConfigSNIUsesDomainHost(t *testing.T) {
	opts := Options{
		ServerAddr: "example.com:8443",
	}
	cfg := opts.TLSConfig()
	if cfg.ServerName != "example.com" {
		t.Fatalf("ServerName = %q, want example.com", cfg.ServerName)
	}
}

func TestTLSConfigSNIHidesSNIForIPHost(t *testing.T) {
	opts := Options{
		ServerAddr: "203.0.113.10:8443",
	}
	cfg := opts.TLSConfig()
	if cfg.ServerName != "" {
		t.Fatalf("ServerName = %q, want empty (no SNI for IP)", cfg.ServerName)
	}
}

func TestTLSConfigCertPinVerification(t *testing.T) {
	opts := Options{
		ServerAddr:    "example.com:8443",
		TLSCertSHA256: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	}
	cfg := opts.TLSConfig()
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
}

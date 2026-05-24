package util

import (
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestGenerateKeyPairAddsIPSAN(t *testing.T) {
	cert, err := GenerateKeyPair(func() time.Time { return time.Unix(100, 0) }, "203.0.113.10")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.IPAddresses) != 1 || !parsed.IPAddresses[0].Equal(net.ParseIP("203.0.113.10")) {
		t.Fatalf("unexpected IP SANs: %#v", parsed.IPAddresses)
	}
	if len(parsed.DNSNames) != 0 {
		t.Fatalf("unexpected DNS SANs: %#v", parsed.DNSNames)
	}
}

func TestGenerateKeyPairAddsDNSSAN(t *testing.T) {
	cert, err := GenerateKeyPair(func() time.Time { return time.Unix(100, 0) }, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.DNSNames) != 1 || parsed.DNSNames[0] != "example.com" {
		t.Fatalf("unexpected DNS SANs: %#v", parsed.DNSNames)
	}
	if len(parsed.IPAddresses) != 0 {
		t.Fatalf("unexpected IP SANs: %#v", parsed.IPAddresses)
	}
}

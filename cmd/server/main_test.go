package main

import "testing"

func TestGeneratedCertificateNameDefaultsUnspecifiedListenToLocalhost(t *testing.T) {
	got, err := generatedIPCertificateName("", "0.0.0.0:8443")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1" {
		t.Fatalf("generatedCertificateName() = %q, want 127.0.0.1", got)
	}
}

func TestGeneratedCertificateNameUsesListenIP(t *testing.T) {
	got, err := generatedIPCertificateName("", "203.0.113.10:8443")
	if err != nil {
		t.Fatal(err)
	}
	if got != "203.0.113.10" {
		t.Fatalf("generatedCertificateName() = %q, want 203.0.113.10", got)
	}
}

func TestGeneratedCertificateNameRejectsDomainForIPCert(t *testing.T) {
	if _, err := generatedIPCertificateName("example.com", "0.0.0.0:8443"); err == nil {
		t.Fatal("expected domain name to be rejected for ip certificate")
	}
}

func TestACMEDomainNameAcceptsDomain(t *testing.T) {
	got, err := acmeDomainName("Example.COM")
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com" {
		t.Fatalf("acmeDomainName() = %q, want example.com", got)
	}
}

func TestACMEDomainNameRejectsIP(t *testing.T) {
	if _, err := acmeDomainName("203.0.113.10"); err == nil {
		t.Fatal("expected IP address to be rejected for domain certificate")
	}
}

func TestACMEDomainNameRejectsHostPort(t *testing.T) {
	if _, err := acmeDomainName("example.com:443"); err == nil {
		t.Fatal("expected host:port to be rejected for domain certificate")
	}
}

func TestGeneratedSelfSignedCertificateNameAcceptsDomainAndIP(t *testing.T) {
	gotDomain, err := generatedSelfSignedCertificateName("example.com", "0.0.0.0:8443", false)
	if err != nil {
		t.Fatal(err)
	}
	if gotDomain != "example.com" {
		t.Fatalf("expected example.com, got %q", gotDomain)
	}

	gotIP, err := generatedSelfSignedCertificateName("127.0.0.1", "0.0.0.0:8443", false)
	if err != nil {
		t.Fatal(err)
	}
	if gotIP != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %q", gotIP)
	}
}

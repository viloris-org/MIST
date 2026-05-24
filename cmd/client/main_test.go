package main

import "testing"

func TestServerNameForTLSUsesExplicitSNI(t *testing.T) {
	got := serverNameForTLS("203.0.113.10", "example.com")
	if got != "example.com" {
		t.Fatalf("serverNameForTLS() = %q, want example.com", got)
	}
}

func TestServerNameForTLSUsesDomainHost(t *testing.T) {
	got := serverNameForTLS("example.com", "")
	if got != "example.com" {
		t.Fatalf("serverNameForTLS() = %q, want example.com", got)
	}
}

func TestServerNameForTLSDisablesSNIForIPHost(t *testing.T) {
	got := serverNameForTLS("203.0.113.10", "")
	if got != "127.0.0.1" {
		t.Fatalf("serverNameForTLS() = %q, want 127.0.0.1", got)
	}
}

func TestParseBoolQuery(t *testing.T) {
	for _, value := range []string{"1", "t", "true", "TRUE"} {
		if !parseBoolQuery(value) {
			t.Fatalf("parseBoolQuery(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"0", "false", "", "yes"} {
		if parseBoolQuery(value) {
			t.Fatalf("parseBoolQuery(%q) = true, want false", value)
		}
	}
}

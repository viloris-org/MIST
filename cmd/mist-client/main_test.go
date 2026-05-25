package main

import "testing"

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

func TestParseInboundSet(t *testing.T) {
	cfg := clientConfig{inbound: "socks,http,redirect"}
	set, err := cfg.parseInboundSet()
	if err != nil {
		t.Fatal(err)
	}
	if !set.socksHTTP || !set.redirect {
		t.Fatalf("parseInboundSet did not enable both listeners: %+v", set)
	}
}

func TestClientConfigCheckValidatesTLSMinVersion(t *testing.T) {
	cfg, err := parseClientConfig([]string{
		"--check",
		"-s", "127.0.0.1:8443",
		"-p", "password",
		"--tls-min-version", "1.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.clientOptions(); err == nil {
		t.Fatal("clientOptions succeeded with invalid TLS min version")
	}
}

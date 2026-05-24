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

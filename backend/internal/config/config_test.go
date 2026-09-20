package config

import "testing"

func TestLoadUsesDefaultHTTPAddress(t *testing.T) {
	t.Setenv("PINERARY_HTTP_ADDR", "")

	cfg := Load()

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected HTTP address %q, got %q", defaultHTTPAddr, cfg.HTTPAddr)
	}
}

func TestLoadUsesConfiguredHTTPAddress(t *testing.T) {
	t.Setenv("PINERARY_HTTP_ADDR", "127.0.0.1:9090")

	cfg := Load()

	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("expected configured HTTP address, got %q", cfg.HTTPAddr)
	}
}

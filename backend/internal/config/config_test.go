package config

import "testing"

func TestLoadUsesDefaultHTTPAddress(t *testing.T) {
	t.Setenv("PINERARY_HTTP_ADDR", "")

	cfg := Load()

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected HTTP address %q, got %q", defaultHTTPAddr, cfg.HTTPAddr)
	}
}

func TestLoadUsesConfiguredDatabaseURL(t *testing.T) {
	t.Setenv("PINERARY_DATABASE_URL", "postgres://example/test")

	cfg := Load()

	if cfg.DatabaseURL != "postgres://example/test" {
		t.Fatalf("expected configured database URL, got %q", cfg.DatabaseURL)
	}
}

func TestLoadUsesConfiguredHTTPAddress(t *testing.T) {
	t.Setenv("PINERARY_HTTP_ADDR", "127.0.0.1:9090")

	cfg := Load()

	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("expected configured HTTP address, got %q", cfg.HTTPAddr)
	}
}

func TestLoadParsesAllowedOrigins(t *testing.T) {
	t.Setenv("PINERARY_ALLOWED_ORIGINS", " https://app.example.com, http://localhost:3000 ,")

	cfg := Load()

	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected two allowed origins, got %#v", cfg.AllowedOrigins)
	}
	if cfg.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("unexpected first allowed origin %q", cfg.AllowedOrigins[0])
	}
}

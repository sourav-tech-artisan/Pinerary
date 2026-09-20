package config

import (
	"os"
	"strings"
)

const (
	defaultAuthMode    = "oidc"
	defaultDatabaseURL = "postgres://pinerary:pinerary@localhost:5432/pinerary?sslmode=disable"
	defaultHTTPAddr    = ":8080"
)

type Config struct {
	AllowedOrigins []string
	AuthMode       string
	DatabaseURL    string
	HTTPAddr       string
	OIDCAudience   string
	OIDCIssuerURL  string
}

func Load() Config {
	httpAddr := os.Getenv("PINERARY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	return Config{
		AllowedOrigins: splitList(os.Getenv("PINERARY_ALLOWED_ORIGINS")),
		AuthMode:       valueOrDefault("PINERARY_AUTH_MODE", defaultAuthMode),
		DatabaseURL:    valueOrDefault("PINERARY_DATABASE_URL", defaultDatabaseURL),
		HTTPAddr:       httpAddr,
		OIDCAudience:   os.Getenv("PINERARY_OIDC_AUDIENCE"),
		OIDCIssuerURL:  os.Getenv("PINERARY_OIDC_ISSUER_URL"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

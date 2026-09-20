package config

import (
	"os"
	"strings"
)

const (
	defaultDatabaseURL = "postgres://pinerary:pinerary@localhost:5432/pinerary?sslmode=disable"
	defaultHTTPAddr    = ":8080"
)

type Config struct {
	AllowedOrigins []string
	DatabaseURL    string
	HTTPAddr       string
}

func Load() Config {
	httpAddr := os.Getenv("PINERARY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	return Config{
		AllowedOrigins: splitList(os.Getenv("PINERARY_ALLOWED_ORIGINS")),
		DatabaseURL:    valueOrDefault("PINERARY_DATABASE_URL", defaultDatabaseURL),
		HTTPAddr:       httpAddr,
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

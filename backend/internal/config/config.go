package config

import "os"

const defaultHTTPAddr = ":8080"

type Config struct {
	HTTPAddr string
}

func Load() Config {
	httpAddr := os.Getenv("PINERARY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	return Config{HTTPAddr: httpAddr}
}

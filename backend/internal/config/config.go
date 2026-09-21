package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultAuthMode                 = "oidc"
	defaultDatabaseURL              = "postgres://pinerary:pinerary@localhost:5432/pinerary?sslmode=disable"
	defaultHTTPAddr                 = ":8080"
	defaultNominatimURL             = "https://nominatim.openstreetmap.org"
	defaultNominatimUserAgent       = "Pinerary/0.1"
	defaultValhallaURL              = "http://localhost:8002"
	defaultObjectEndpoint           = "localhost:9000"
	defaultObjectAccessKey          = "pinerary"
	defaultObjectSecretKey          = "pinerary-secret"
	defaultObjectBucket             = "pinerary-media"
	defaultMaxPhotoBytes      int64 = 15 << 20
	defaultPublicBaseURL            = "http://localhost:8080"
)

type Config struct {
	AllowedOrigins     []string
	AuthMode           string
	DatabaseURL        string
	HTTPAddr           string
	NominatimURL       string
	NominatimUserAgent string
	OIDCAudience       string
	OIDCIssuerURL      string
	ValhallaURL        string
	ObjectEndpoint     string
	ObjectAccessKey    string
	ObjectSecretKey    string
	ObjectBucket       string
	ObjectUseTLS       bool
	MaxPhotoBytes      int64
	PublicBaseURL      string
	VAPIDSubscriber    string
	VAPIDPublicKey     string
	VAPIDPrivateKey    string
}

func Load() Config {
	httpAddr := os.Getenv("PINERARY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	return Config{
		AllowedOrigins:     splitList(os.Getenv("PINERARY_ALLOWED_ORIGINS")),
		AuthMode:           valueOrDefault("PINERARY_AUTH_MODE", defaultAuthMode),
		DatabaseURL:        valueOrDefault("PINERARY_DATABASE_URL", defaultDatabaseURL),
		HTTPAddr:           httpAddr,
		NominatimURL:       valueOrDefault("PINERARY_NOMINATIM_URL", defaultNominatimURL),
		NominatimUserAgent: valueOrDefault("PINERARY_NOMINATIM_USER_AGENT", defaultNominatimUserAgent),
		OIDCAudience:       os.Getenv("PINERARY_OIDC_AUDIENCE"),
		OIDCIssuerURL:      os.Getenv("PINERARY_OIDC_ISSUER_URL"),
		ValhallaURL:        valueOrDefault("PINERARY_VALHALLA_URL", defaultValhallaURL),
		ObjectEndpoint:     valueOrDefault("PINERARY_OBJECT_ENDPOINT", defaultObjectEndpoint),
		ObjectAccessKey:    valueOrDefault("PINERARY_OBJECT_ACCESS_KEY", defaultObjectAccessKey),
		ObjectSecretKey:    valueOrDefault("PINERARY_OBJECT_SECRET_KEY", defaultObjectSecretKey),
		ObjectBucket:       valueOrDefault("PINERARY_OBJECT_BUCKET", defaultObjectBucket),
		ObjectUseTLS:       strings.EqualFold(os.Getenv("PINERARY_OBJECT_USE_TLS"), "true"),
		MaxPhotoBytes:      positiveInt64OrDefault("PINERARY_MAX_PHOTO_BYTES", defaultMaxPhotoBytes),
		PublicBaseURL:      strings.TrimRight(valueOrDefault("PINERARY_PUBLIC_BASE_URL", defaultPublicBaseURL), "/"),
		VAPIDSubscriber:    os.Getenv("PINERARY_VAPID_SUBSCRIBER"),
		VAPIDPublicKey:     os.Getenv("PINERARY_VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:    os.Getenv("PINERARY_VAPID_PRIVATE_KEY"),
	}
}

func positiveInt64OrDefault(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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

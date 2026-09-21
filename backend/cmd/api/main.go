package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourav-tech-artisan/Pinerary/backend/internal/config"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/geocoding"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/httpapi"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/journeys"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/media"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/nearby"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/places"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/sharing"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/tracking"
)

const shutdownTimeout = 10 * time.Second

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute
)

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()

	databasePool, err := database.Open(startupContext, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer databasePool.Close()

	verifier, err := buildVerifier(startupContext, cfg)
	if err != nil {
		return err
	}

	queries := dbgen.New(databasePool)
	journeyService := journeys.NewService(queries)
	placeService := places.NewService(databasePool)
	nominatim, err := geocoding.NewNominatimClient(cfg.NominatimURL, cfg.NominatimUserAgent, nil)
	if err != nil {
		return err
	}
	geocodingService := geocoding.NewService(queries, nominatim)
	trackingService := tracking.NewService(databasePool)
	objectStore, err := objectstore.NewMinIOStore(
		cfg.ObjectEndpoint, cfg.ObjectAccessKey, cfg.ObjectSecretKey, cfg.ObjectBucket, cfg.ObjectUseTLS,
	)
	if err != nil {
		return err
	}
	if err := objectStore.EnsureBucket(startupContext); err != nil {
		return err
	}
	mediaService := media.NewService(databasePool, objectStore, cfg.MaxPhotoBytes)
	valhalla, err := routing.NewValhallaClient(cfg.ValhallaURL, nil)
	if err != nil {
		return err
	}
	nearbyService := nearby.NewService(queries, valhalla)
	sharingService := sharing.NewService(queries, objectStore, cfg.PublicBaseURL)
	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.RouterConfig{
			AllowedOrigins:   cfg.AllowedOrigins,
			Logger:           logger,
			Ready:            databasePool.Ping,
			UserProvisioner:  queries,
			ProfileStore:     queries,
			JourneyService:   journeyService,
			PlaceService:     placeService,
			GeocodingService: geocodingService,
			TrackingService:  trackingService,
			MediaService:     mediaService,
			NearbyService:    nearbyService,
			SharingService:   sharingService,
			Verifier:         verifier,
		}),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("starting API", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signalContext.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("shutting down API")
	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}

	return nil
}

func buildVerifier(ctx context.Context, cfg config.Config) (identity.Verifier, error) {
	switch cfg.AuthMode {
	case "oidc":
		return identity.NewOIDCVerifier(ctx, cfg.OIDCIssuerURL, cfg.OIDCAudience)
	case "development":
		return identity.DevelopmentVerifier{}, nil
	default:
		return nil, fmt.Errorf("unsupported authentication mode %q", cfg.AuthMode)
	}
}

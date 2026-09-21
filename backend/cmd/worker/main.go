package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourav-tech-artisan/Pinerary/backend/internal/config"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/jobs"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/media"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/notifications"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/telemetry"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/tracking"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	telemetryShutdown, err := telemetry.Setup(ctx, "pinerary-worker", cfg.OTLPEndpoint)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = telemetryShutdown(shutdownContext)
	}()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	workerID := fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().Unix())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	runner := jobs.NewRunner(dbgen.New(pool), logger, workerID)
	trackProcessor := tracking.NewProcessor(pool)
	runner.Register("track.process", trackProcessor.HandleJob)
	valhalla, err := routing.NewValhallaClient(cfg.ValhallaURL, nil)
	if err != nil {
		return err
	}
	trackMatcher := tracking.NewMatcher(dbgen.New(pool), valhalla)
	runner.Register("track.match", trackMatcher.HandleJob)
	objectStore, err := objectstore.NewMinIOStore(
		cfg.ObjectEndpoint, cfg.ObjectAccessKey, cfg.ObjectSecretKey, cfg.ObjectBucket, cfg.ObjectUseTLS,
	)
	if err != nil {
		return err
	}
	photoProcessor := media.NewProcessor(dbgen.New(pool), objectStore, cfg.MaxPhotoBytes)
	runner.Register("photo.process", photoProcessor.HandleJob)
	pushSender := notifications.NewWebPushSender(cfg.VAPIDSubscriber, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey)
	lifecycleProcessor := notifications.NewLifecycleProcessor(dbgen.New(pool), pushSender)
	runner.Register("outing.warn", lifecycleProcessor.HandleWarning)
	runner.Register("outing.expire", lifecycleProcessor.HandleExpiry)

	logger.Info("starting worker", "worker_id", workerID)
	return runner.Run(ctx)
}

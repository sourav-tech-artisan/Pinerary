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

	pool, err := database.Open(ctx, config.Load().DatabaseURL)
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

	logger.Info("starting worker", "worker_id", workerID)
	return runner.Run(ctx)
}

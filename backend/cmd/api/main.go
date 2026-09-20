package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourav-tech-artisan/Pinerary/backend/internal/config"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/httpapi"
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

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(httpapi.RouterConfig{AllowedOrigins: cfg.AllowedOrigins, Logger: logger}),
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

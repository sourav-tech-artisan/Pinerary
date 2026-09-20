package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourav-tech-artisan/Pinerary/backend/internal/config"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Printf("API stopped with an error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpapi.NewRouter(),
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("starting API on %s", cfg.HTTPAddr)
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

	log.Print("shutting down API")
	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}

	return nil
}

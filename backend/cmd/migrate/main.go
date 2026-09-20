package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/config"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/migrations"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: migrate <up|down|status>")
	}

	db, err := sql.Open("pgx", config.Load().DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure goose: %w", err)
	}

	switch args[0] {
	case "up":
		err = goose.UpContext(ctx, db, ".")
	case "down":
		err = goose.DownContext(ctx, db, ".")
	case "status":
		err = goose.StatusContext(ctx, db, ".")
	default:
		return errors.New("usage: migrate <up|down|status>")
	}
	if err != nil {
		return fmt.Errorf("run %s migration: %w", args[0], err)
	}

	return nil
}

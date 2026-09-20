package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

// WithinTransaction runs fn with generated queries bound to one transaction.
// It rolls back automatically unless fn and the final commit both succeed.
func WithinTransaction(
	ctx context.Context,
	pool *pgxpool.Pool,
	fn func(*dbgen.Queries) error,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is intentionally harmless

	if err := fn(dbgen.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

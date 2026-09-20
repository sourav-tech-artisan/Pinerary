package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type Handler func(context.Context, []byte) error

type Store interface {
	ClaimNextJobs(context.Context, dbgen.ClaimNextJobsParams) ([]dbgen.BackgroundJob, error)
	CompleteJob(context.Context, int64) error
	RequeueStaleJobs(context.Context, pgtype.Timestamptz) (int64, error)
	RetryJob(context.Context, dbgen.RetryJobParams) error
}

type Runner struct {
	store        Store
	handlers     map[string]Handler
	logger       *slog.Logger
	workerID     string
	pollInterval time.Duration
	batchSize    int32
}

func NewRunner(store Store, logger *slog.Logger, workerID string) *Runner {
	return &Runner{
		store:        store,
		handlers:     make(map[string]Handler),
		logger:       logger,
		workerID:     workerID,
		pollInterval: time.Second,
		batchSize:    10,
	}
}

func (r *Runner) Register(jobType string, handler Handler) {
	r.handlers[jobType] = handler
}

func (r *Runner) Run(ctx context.Context) error {
	staleBefore := pgtype.Timestamptz{Time: time.Now().Add(-15 * time.Minute), Valid: true}
	if _, err := r.store.RequeueStaleJobs(ctx, staleBefore); err != nil {
		return fmt.Errorf("requeue stale jobs: %w", err)
	}

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if err := r.work(ctx); err != nil && !errors.Is(err, context.Canceled) {
			r.logger.ErrorContext(ctx, "job polling failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Runner) work(ctx context.Context) error {
	claimed, err := r.store.ClaimNextJobs(ctx, dbgen.ClaimNextJobsParams{
		BatchSize: r.batchSize,
		WorkerID:  pgtype.Text{String: r.workerID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("claim jobs: %w", err)
	}

	for _, job := range claimed {
		r.handle(ctx, job)
	}
	return nil
}

func (r *Runner) handle(ctx context.Context, job dbgen.BackgroundJob) {
	handler, ok := r.handlers[job.JobType]
	if !ok {
		r.retry(ctx, job, fmt.Errorf("no handler registered for %q", job.JobType))
		return
	}

	if err := handler(ctx, job.Payload); err != nil {
		r.retry(ctx, job, err)
		return
	}

	if err := r.store.CompleteJob(ctx, job.ID); err != nil {
		r.logger.ErrorContext(ctx, "complete job failed", "job_id", job.ID, "error", err)
	}
}

func (r *Runner) retry(ctx context.Context, job dbgen.BackgroundJob, jobErr error) {
	delay := time.Duration(math.Pow(2, float64(min(job.Attempts, 8)))) * time.Second
	errorMessage := jobErr.Error()
	if len(errorMessage) > 2000 {
		errorMessage = errorMessage[:2000]
	}

	err := r.store.RetryJob(ctx, dbgen.RetryJobParams{
		ID:        job.ID,
		RunAt:     pgtype.Timestamptz{Time: time.Now().Add(delay), Valid: true},
		LastError: pgtype.Text{String: errorMessage, Valid: true},
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "retry job failed", "job_id", job.ID, "error", err)
	}
}

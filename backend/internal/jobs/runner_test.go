package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type fakeStore struct {
	jobs      []dbgen.BackgroundJob
	completed []int64
	retried   []dbgen.RetryJobParams
}

func (s *fakeStore) ClaimNextJobs(context.Context, dbgen.ClaimNextJobsParams) ([]dbgen.BackgroundJob, error) {
	return s.jobs, nil
}

func (s *fakeStore) CompleteJob(_ context.Context, id int64) error {
	s.completed = append(s.completed, id)
	return nil
}

func (s *fakeStore) RequeueStaleJobs(context.Context, pgtype.Timestamptz) (int64, error) {
	return 0, nil
}

func (s *fakeStore) RetryJob(_ context.Context, params dbgen.RetryJobParams) error {
	s.retried = append(s.retried, params)
	return nil
}

func TestWorkCompletesSuccessfulJob(t *testing.T) {
	store := &fakeStore{jobs: []dbgen.BackgroundJob{{ID: 42, JobType: "test", Payload: []byte(`{"ok":true}`)}}}
	runner := NewRunner(store, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker-test")
	runner.Register("test", func(context.Context, []byte) error { return nil })

	if err := runner.work(context.Background()); err != nil {
		t.Fatalf("work() error = %v", err)
	}
	if len(store.completed) != 1 || store.completed[0] != 42 {
		t.Fatalf("completed = %#v", store.completed)
	}
}

func TestWorkRetriesFailedJob(t *testing.T) {
	store := &fakeStore{jobs: []dbgen.BackgroundJob{{ID: 7, JobType: "test", Attempts: 1}}}
	runner := NewRunner(store, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker-test")
	runner.Register("test", func(context.Context, []byte) error { return errors.New("temporary failure") })

	if err := runner.work(context.Background()); err != nil {
		t.Fatalf("work() error = %v", err)
	}
	if len(store.retried) != 1 || store.retried[0].ID != 7 {
		t.Fatalf("retried = %#v", store.retried)
	}
}

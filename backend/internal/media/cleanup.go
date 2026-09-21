package media

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
)

const (
	abandonedUploadAge = 24 * time.Hour
	cleanupInterval    = time.Hour
	cleanupBatchSize   = 100
)

type cleanupStore interface {
	ClaimAbandonedPhotoUploads(context.Context, dbgen.ClaimAbandonedPhotoUploadsParams) ([]dbgen.ClaimAbandonedPhotoUploadsRow, error)
	DeleteAbandonedPhotoUpload(context.Context, dbgen.DeleteAbandonedPhotoUploadParams) (int64, error)
	EnqueueJob(context.Context, dbgen.EnqueueJobParams) (dbgen.BackgroundJob, error)
}

type CleanupProcessor struct {
	database cleanupStore
	objects  objectstore.Deleter
	now      func() time.Time
}

func NewCleanupProcessor(database cleanupStore, objects objectstore.Deleter) *CleanupProcessor {
	return &CleanupProcessor{database: database, objects: objects, now: time.Now}
}

func (p *CleanupProcessor) Schedule(ctx context.Context, runAt time.Time) error {
	payload, _ := json.Marshal(map[string]int{"version": 1})
	_, err := p.database.EnqueueJob(ctx, dbgen.EnqueueJobParams{
		JobType: "photo.cleanup", Payload: payload,
		IdempotencyKey: pgtype.Text{String: "photo.cleanup:" + runAt.UTC().Truncate(cleanupInterval).Format(time.RFC3339), Valid: true},
		MaxAttempts:    8,
		RunAt:          pgtype.Timestamptz{Time: runAt.UTC(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("schedule photo cleanup: %w", err)
	}
	return nil
}

func (p *CleanupProcessor) HandleJob(ctx context.Context, _ []byte) error {
	now := p.now().UTC()
	cutoff := now.Add(-abandonedUploadAge)
	photos, err := p.database.ClaimAbandonedPhotoUploads(ctx, dbgen.ClaimAbandonedPhotoUploadsParams{
		CreatedBefore: pgtype.Timestamptz{Time: cutoff, Valid: true}, BatchSize: cleanupBatchSize,
	})
	if err != nil {
		return fmt.Errorf("claim abandoned photo uploads: %w", err)
	}
	for _, photo := range photos {
		if err := p.objects.Delete(ctx, photo.ObjectKey); err != nil {
			return err
		}
		if _, err := p.database.DeleteAbandonedPhotoUpload(ctx, dbgen.DeleteAbandonedPhotoUploadParams{
			ID: photo.ID, CreatedBefore: pgtype.Timestamptz{Time: cutoff, Valid: true},
		}); err != nil {
			return fmt.Errorf("delete abandoned photo upload: %w", err)
		}
	}
	return p.Schedule(ctx, now.Add(cleanupInterval))
}

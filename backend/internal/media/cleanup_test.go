package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type fakeCleanupStore struct {
	photos    []dbgen.ClaimAbandonedPhotoUploadsRow
	deleted   []dbgen.DeleteAbandonedPhotoUploadParams
	scheduled []dbgen.EnqueueJobParams
}

func (s *fakeCleanupStore) DeleteAbandonedPhotoUpload(
	_ context.Context,
	params dbgen.DeleteAbandonedPhotoUploadParams,
) (int64, error) {
	s.deleted = append(s.deleted, params)
	return 1, nil
}

func (s *fakeCleanupStore) EnqueueJob(_ context.Context, params dbgen.EnqueueJobParams) (dbgen.BackgroundJob, error) {
	s.scheduled = append(s.scheduled, params)
	return dbgen.BackgroundJob{}, nil
}

func (s *fakeCleanupStore) ClaimAbandonedPhotoUploads(
	context.Context,
	dbgen.ClaimAbandonedPhotoUploadsParams,
) ([]dbgen.ClaimAbandonedPhotoUploadsRow, error) {
	return s.photos, nil
}

type fakeObjectDeleter struct {
	keys []string
}

func (d *fakeObjectDeleter) Delete(_ context.Context, key string) error {
	d.keys = append(d.keys, key)
	return nil
}

func TestCleanupProcessorDeletesPendingObjectsAndSchedulesNextRun(t *testing.T) {
	now := time.Date(2026, time.September, 21, 12, 30, 0, 0, time.UTC)
	photoID := uuid.New()
	store := &fakeCleanupStore{photos: []dbgen.ClaimAbandonedPhotoUploadsRow{{
		ID: pgtype.UUID{Bytes: photoID, Valid: true}, ObjectKey: "users/test/photos/test/original",
	}}}
	objects := &fakeObjectDeleter{}
	processor := NewCleanupProcessor(store, objects)
	processor.now = func() time.Time { return now }

	if err := processor.HandleJob(context.Background(), nil); err != nil {
		t.Fatalf("HandleJob() error = %v", err)
	}
	if len(objects.keys) != 1 || objects.keys[0] != "users/test/photos/test/original" {
		t.Fatalf("deleted object keys = %#v", objects.keys)
	}
	if len(store.deleted) != 1 || store.deleted[0].ID.Bytes != photoID {
		t.Fatalf("deleted rows = %#v", store.deleted)
	}
	if got := store.deleted[0].CreatedBefore.Time; !got.Equal(now.Add(-abandonedUploadAge)) {
		t.Fatalf("cleanup cutoff = %v", got)
	}
	if len(store.scheduled) != 1 || !store.scheduled[0].RunAt.Time.Equal(now.Add(cleanupInterval)) {
		t.Fatalf("scheduled jobs = %#v", store.scheduled)
	}
}

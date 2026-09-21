package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
)

const (
	uploadExpiry           = 15 * time.Minute
	privatePhotoReadExpiry = 15 * time.Minute
)

var (
	ErrInvalidPhoto  = errors.New("photo metadata is invalid")
	ErrNotFound      = errors.New("photo, journey, or stop not found")
	ErrUploadInvalid = errors.New("uploaded object does not match the reservation")
	ErrUploadState   = errors.New("photo upload cannot be completed in its current state")
)

type Service struct {
	pool         *pgxpool.Pool
	store        objectstore.Store
	maxPhotoSize int64
	now          func() time.Time
}

type ReserveInput struct {
	OwnerID         uuid.UUID
	JourneyID       uuid.UUID
	StopID          *uuid.UUID
	ClientRequestID uuid.UUID
	ContentType     string
	CapturedAt      *time.Time
}

type UploadIntent struct {
	PhotoID     uuid.UUID `json:"photo_id"`
	UploadURL   string    `json:"upload_url"`
	Method      string    `json:"method"`
	ExpiresAt   time.Time `json:"expires_at"`
	MaxBytes    int64     `json:"max_bytes"`
	ContentType string    `json:"content_type"`
}

type CompleteResult struct {
	PhotoID uuid.UUID `json:"photo_id"`
	Status  string    `json:"status"`
}

type PhotoView struct {
	PhotoID    uuid.UUID  `json:"photo_id"`
	URL        string     `json:"url"`
	CapturedAt *time.Time `json:"captured_at"`
}

func NewService(pool *pgxpool.Pool, store objectstore.Store, maxPhotoSize int64) *Service {
	return &Service{pool: pool, store: store, maxPhotoSize: maxPhotoSize, now: time.Now}
}

func (s *Service) Reserve(ctx context.Context, input ReserveInput) (UploadIntent, error) {
	if input.OwnerID == uuid.Nil || input.JourneyID == uuid.Nil || input.ClientRequestID == uuid.Nil ||
		!allowedContentType(input.ContentType) {
		return UploadIntent{}, ErrInvalidPhoto
	}
	photoID := uuid.New()
	objectKey := path.Join("users", input.OwnerID.String(), "photos", photoID.String(), "original")

	var stopID pgtype.UUID
	if input.StopID != nil {
		stopID = pgUUID(*input.StopID)
	}
	var capturedAt pgtype.Timestamptz
	if input.CapturedAt != nil {
		capturedAt = pgtype.Timestamptz{Time: input.CapturedAt.UTC(), Valid: true}
	}
	photo, err := dbgen.New(s.pool).CreatePhotoUpload(ctx, dbgen.CreatePhotoUploadParams{
		ID: pgUUID(photoID), OwnerID: pgUUID(input.OwnerID), JourneyID: pgUUID(input.JourneyID),
		StopID: stopID, ClientRequestID: pgUUID(input.ClientRequestID), ObjectKey: objectKey,
		ContentType: input.ContentType, CapturedAt: capturedAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return UploadIntent{}, ErrNotFound
	}
	if err != nil {
		return UploadIntent{}, fmt.Errorf("create photo reservation: %w", err)
	}

	expiresAt := s.now().Add(uploadExpiry)
	uploadURL, err := s.store.PresignPut(ctx, photo.ObjectKey, uploadExpiry)
	if err != nil {
		return UploadIntent{}, err
	}
	return UploadIntent{
		PhotoID: uuid.UUID(photo.ID.Bytes), UploadURL: uploadURL.String(), Method: "PUT",
		ExpiresAt: expiresAt, MaxBytes: s.maxPhotoSize, ContentType: photo.ContentType,
	}, nil
}

func (s *Service) Complete(ctx context.Context, ownerID, photoID uuid.UUID, checksum string) (CompleteResult, error) {
	queries := dbgen.New(s.pool)
	photo, err := queries.GetPhotoForOwner(ctx, dbgen.GetPhotoForOwnerParams{ID: pgUUID(photoID), OwnerID: pgUUID(ownerID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return CompleteResult{}, ErrNotFound
	}
	if err != nil {
		return CompleteResult{}, fmt.Errorf("get photo upload: %w", err)
	}
	if photo.Status == "processed" {
		return CompleteResult{PhotoID: photoID, Status: photo.Status}, nil
	}
	if photo.Status != "pending" && photo.Status != "uploaded" {
		return CompleteResult{}, ErrUploadState
	}

	object, err := s.store.Stat(ctx, photo.ObjectKey)
	if err != nil {
		return CompleteResult{}, err
	}
	if object.Size <= 0 || object.Size > s.maxPhotoSize ||
		(object.ContentType != "" && object.ContentType != photo.ContentType) {
		return CompleteResult{}, ErrUploadInvalid
	}

	err = database.WithinTransaction(ctx, s.pool, func(tx *dbgen.Queries) error {
		_, err := tx.CompletePhotoUpload(ctx, dbgen.CompletePhotoUploadParams{
			ID: pgUUID(photoID), OwnerID: pgUUID(ownerID),
			ByteSize:       pgtype.Int8{Int64: object.Size, Valid: true},
			ChecksumSha256: pgtype.Text{String: checksum, Valid: checksum != ""},
		})
		if err != nil {
			return fmt.Errorf("mark photo uploaded: %w", err)
		}
		jobPayload, _ := json.Marshal(map[string]uuid.UUID{"photo_id": photoID})
		_, err = tx.EnqueueJob(ctx, dbgen.EnqueueJobParams{
			JobType: "photo.process", Payload: jobPayload,
			IdempotencyKey: pgtype.Text{String: photoID.String(), Valid: true},
			MaxAttempts:    5,
			RunAt:          pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("enqueue photo processing: %w", err)
		}
		return nil
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{PhotoID: photoID, Status: "uploaded"}, nil
}

func (s *Service) ListStopPhotos(ctx context.Context, ownerID, journeyID, stopID uuid.UUID) ([]PhotoView, error) {
	photos, err := dbgen.New(s.pool).ListStopPhotos(ctx, dbgen.ListStopPhotosParams{
		OwnerID: pgUUID(ownerID), JourneyID: pgUUID(journeyID), StopID: pgUUID(stopID),
	})
	if err != nil {
		return nil, fmt.Errorf("list stop photos: %w", err)
	}
	result := make([]PhotoView, 0, len(photos))
	for _, photo := range photos {
		if !photo.ThumbnailKey.Valid {
			continue
		}
		photoURL, err := s.store.PresignGet(ctx, photo.ThumbnailKey.String, privatePhotoReadExpiry)
		if err != nil {
			return nil, err
		}
		view := PhotoView{PhotoID: uuid.UUID(photo.ID.Bytes), URL: photoURL.String()}
		if photo.CapturedAt.Valid {
			capturedAt := photo.CapturedAt.Time
			view.CapturedAt = &capturedAt
		}
		result = append(result, view)
	}
	return result, nil
}

func allowedContentType(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

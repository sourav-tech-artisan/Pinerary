package sharing

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
)

const publicPhotoExpiry = 30 * time.Minute

var (
	ErrInvalidShare = errors.New("share selection contains an invalid stop or photo")
	ErrNotFound     = errors.New("share or journey not found")
)

type Service struct {
	store         *dbgen.Queries
	objects       objectstore.Store
	publicBaseURL string
}

type ItemInput struct {
	StopID      uuid.UUID   `json:"stop_id"`
	DisplayName *string     `json:"display_name"`
	Note        *string     `json:"note"`
	PhotoIDs    []uuid.UUID `json:"photo_ids"`
}

type CreateInput struct {
	OwnerID   uuid.UUID
	JourneyID uuid.UUID
	Title     string
	Items     []ItemInput
	ExpiresAt *time.Time
}

type Snapshot struct {
	Title     string         `json:"title"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   *time.Time     `json:"ended_at"`
	Items     []SnapshotItem `json:"items"`
}

type SnapshotItem struct {
	StopID      uuid.UUID       `json:"stop_id"`
	DisplayName string          `json:"display_name"`
	Note        string          `json:"note"`
	Latitude    float64         `json:"latitude"`
	Longitude   float64         `json:"longitude"`
	CapturedAt  time.Time       `json:"captured_at"`
	Photos      []SnapshotPhoto `json:"photos"`
}

type SnapshotPhoto struct {
	ID           uuid.UUID `json:"id"`
	ThumbnailKey string    `json:"thumbnail_key"`
}

type PublicSnapshot struct {
	Title     string       `json:"title"`
	StartedAt time.Time    `json:"started_at"`
	EndedAt   *time.Time   `json:"ended_at"`
	Items     []PublicItem `json:"items"`
}

type PublicItem struct {
	StopID      uuid.UUID     `json:"stop_id"`
	DisplayName string        `json:"display_name"`
	Note        string        `json:"note"`
	Latitude    float64       `json:"latitude"`
	Longitude   float64       `json:"longitude"`
	CapturedAt  time.Time     `json:"captured_at"`
	Photos      []PublicPhoto `json:"photos"`
}

type PublicPhoto struct {
	ID  uuid.UUID `json:"id"`
	URL string    `json:"url"`
}

type CreatedShare struct {
	ID      uuid.UUID `json:"id"`
	URL     string    `json:"url"`
	Message string    `json:"message"`
}

func NewService(store *dbgen.Queries, objects objectstore.Store, publicBaseURL string) *Service {
	return &Service{store: store, objects: objects, publicBaseURL: publicBaseURL}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedShare, error) {
	if len([]rune(input.Title)) > 160 || len(input.Items) > 200 ||
		(input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now())) {
		return CreatedShare{}, ErrInvalidShare
	}
	for _, item := range input.Items {
		if (item.DisplayName != nil && len([]rune(*item.DisplayName)) > 200) ||
			(item.Note != nil && len([]rune(*item.Note)) > 5000) || len(item.PhotoIDs) > 20 {
			return CreatedShare{}, ErrInvalidShare
		}
	}
	journey, err := s.store.GetJourney(ctx, dbgen.GetJourneyParams{
		ID: pgUUID(input.JourneyID), OwnerID: pgUUID(input.OwnerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return CreatedShare{}, ErrNotFound
	}
	if err != nil {
		return CreatedShare{}, fmt.Errorf("get journey for share: %w", err)
	}
	stops, err := s.store.ListJourneyStops(ctx, dbgen.ListJourneyStopsParams{
		JourneyID: pgUUID(input.JourneyID), OwnerID: pgUUID(input.OwnerID),
	})
	if err != nil {
		return CreatedShare{}, fmt.Errorf("list stops for share: %w", err)
	}
	photos, err := s.store.ListPhotosForShare(ctx, dbgen.ListPhotosForShareParams{
		OwnerID: pgUUID(input.OwnerID), JourneyID: pgUUID(input.JourneyID),
	})
	if err != nil {
		return CreatedShare{}, fmt.Errorf("list photos for share: %w", err)
	}

	stopByID := make(map[uuid.UUID]dbgen.ListJourneyStopsRow, len(stops))
	for _, stop := range stops {
		stopByID[uuid.UUID(stop.ID.Bytes)] = stop
	}
	photoByID := make(map[uuid.UUID]dbgen.ListPhotosForShareRow, len(photos))
	for _, photo := range photos {
		photoByID[uuid.UUID(photo.ID.Bytes)] = photo
	}
	if len(input.Items) == 0 {
		for _, stop := range stops {
			input.Items = append(input.Items, ItemInput{StopID: uuid.UUID(stop.ID.Bytes)})
		}
	}

	snapshot := Snapshot{Title: input.Title, StartedAt: journey.StartedAt.Time}
	if snapshot.Title == "" {
		snapshot.Title = journey.Label
	}
	if journey.EndedAt.Valid {
		endedAt := journey.EndedAt.Time
		snapshot.EndedAt = &endedAt
	}
	seenStops := make(map[uuid.UUID]struct{}, len(input.Items))
	for _, item := range input.Items {
		stop, ok := stopByID[item.StopID]
		if !ok {
			return CreatedShare{}, ErrInvalidShare
		}
		if _, duplicate := seenStops[item.StopID]; duplicate {
			return CreatedShare{}, ErrInvalidShare
		}
		seenStops[item.StopID] = struct{}{}
		snapshotItem := SnapshotItem{
			StopID: item.StopID, DisplayName: stop.DisplayName, Note: stop.Note,
			Latitude: stop.Latitude, Longitude: stop.Longitude, CapturedAt: stop.CapturedAt.Time,
			Photos: []SnapshotPhoto{},
		}
		if item.DisplayName != nil {
			snapshotItem.DisplayName = *item.DisplayName
		}
		if item.Note != nil {
			snapshotItem.Note = *item.Note
		}
		for _, photoID := range item.PhotoIDs {
			photo, ok := photoByID[photoID]
			if !ok || !photo.StopID.Valid || uuid.UUID(photo.StopID.Bytes) != item.StopID || !photo.ThumbnailKey.Valid {
				return CreatedShare{}, ErrInvalidShare
			}
			snapshotItem.Photos = append(snapshotItem.Photos, SnapshotPhoto{ID: photoID, ThumbnailKey: photo.ThumbnailKey.String})
		}
		snapshot.Items = append(snapshot.Items, snapshotItem)
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return CreatedShare{}, fmt.Errorf("encode share snapshot: %w", err)
	}

	token, tokenHash, err := newToken()
	if err != nil {
		return CreatedShare{}, err
	}
	var expiresAt pgtype.Timestamptz
	if input.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: input.ExpiresAt.UTC(), Valid: true}
	}
	share, err := s.store.CreateItineraryShare(ctx, dbgen.CreateItineraryShareParams{
		JourneyID: pgUUID(input.JourneyID), OwnerID: pgUUID(input.OwnerID),
		TokenHash: tokenHash, Snapshot: snapshotJSON, ExpiresAt: expiresAt,
	})
	if err != nil {
		return CreatedShare{}, fmt.Errorf("create itinerary share: %w", err)
	}
	shareURL := s.publicBaseURL + "/s/" + token
	return CreatedShare{
		ID: uuid.UUID(share.ID.Bytes), URL: shareURL,
		Message: snapshot.Title + "\nHere is my itinerary:\n" + shareURL,
	}, nil
}

func (s *Service) Resolve(ctx context.Context, token string) (PublicSnapshot, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return PublicSnapshot{}, ErrNotFound
	}
	hash := sha256.Sum256(decoded)
	share, err := s.store.GetItineraryShareByTokenHash(ctx, hash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicSnapshot{}, ErrNotFound
	}
	if err != nil {
		return PublicSnapshot{}, fmt.Errorf("get itinerary share: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(share.Snapshot, &snapshot); err != nil {
		return PublicSnapshot{}, fmt.Errorf("decode itinerary share: %w", err)
	}
	public := PublicSnapshot{Title: snapshot.Title, StartedAt: snapshot.StartedAt, EndedAt: snapshot.EndedAt}
	for _, item := range snapshot.Items {
		publicItem := PublicItem{
			StopID: item.StopID, DisplayName: item.DisplayName, Note: item.Note,
			Latitude: item.Latitude, Longitude: item.Longitude, CapturedAt: item.CapturedAt,
			Photos: []PublicPhoto{},
		}
		for _, photo := range item.Photos {
			photoURL, err := s.objects.PresignGet(ctx, photo.ThumbnailKey, publicPhotoExpiry)
			if err != nil {
				return PublicSnapshot{}, err
			}
			publicItem.Photos = append(publicItem.Photos, PublicPhoto{ID: photo.ID, URL: photoURL.String()})
		}
		public.Items = append(public.Items, publicItem)
	}
	return public, nil
}

func (s *Service) Revoke(ctx context.Context, ownerID, shareID uuid.UUID) error {
	count, err := s.store.RevokeItineraryShare(ctx, dbgen.RevokeItineraryShareParams{
		ID: pgUUID(shareID), OwnerID: pgUUID(ownerID),
	})
	if err != nil {
		return fmt.Errorf("revoke itinerary share: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func newToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate share token: %w", err)
	}
	hash := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), hash[:], nil
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

package tracking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

const MaxBatchSize = 500

type Point struct {
	SampleID   uuid.UUID `json:"sample_id"`
	CapturedAt time.Time `json:"captured_at"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	AccuracyM  *float32  `json:"accuracy_m"`
	SpeedMPS   *float32  `json:"speed_mps"`
	Heading    *float32  `json:"heading_deg"`
}

type classifiedPoint struct {
	Point
	IsAccepted      bool    `json:"is_accepted"`
	RejectionReason *string `json:"rejection_reason"`
}

type IngestResult struct {
	Received   int `json:"received"`
	Inserted   int `json:"inserted"`
	Duplicates int `json:"duplicates"`
	Flagged    int `json:"flagged"`
}

type Service struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

var (
	ErrInvalidBatch     = errors.New("location batch must contain 1 to 500 valid points")
	ErrJourneyNotActive = errors.New("journey is not active")
)

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

func (s *Service) Ingest(ctx context.Context, ownerID, journeyID uuid.UUID, points []Point) (IngestResult, error) {
	if len(points) == 0 || len(points) > MaxBatchSize {
		return IngestResult{}, ErrInvalidBatch
	}

	classified := make([]classifiedPoint, 0, len(points))
	flagged := 0
	for _, point := range points {
		result, err := s.classify(point)
		if err != nil {
			return IngestResult{}, err
		}
		if !result.IsAccepted {
			flagged++
		}
		classified = append(classified, result)
	}
	payload, err := json.Marshal(classified)
	if err != nil {
		return IngestResult{}, fmt.Errorf("encode location batch: %w", err)
	}

	inserted := int64(0)
	err = database.WithinTransaction(ctx, s.pool, func(queries *dbgen.Queries) error {
		_, err := queries.LockActiveJourney(ctx, dbgen.LockActiveJourneyParams{
			ID: pgUUID(journeyID), OwnerID: pgUUID(ownerID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrJourneyNotActive
		}
		if err != nil {
			return fmt.Errorf("lock journey: %w", err)
		}

		inserted, err = queries.InsertLocationSamples(ctx, dbgen.InsertLocationSamplesParams{
			JourneyID: pgUUID(journeyID), UserID: pgUUID(ownerID), Points: payload,
		})
		if err != nil {
			return fmt.Errorf("insert location samples: %w", err)
		}

		lastSampleID := points[len(points)-1].SampleID
		jobPayload, _ := json.Marshal(map[string]uuid.UUID{"journey_id": journeyID})
		_, err = queries.EnqueueJob(ctx, dbgen.EnqueueJobParams{
			JobType: "track.process", Payload: jobPayload,
			IdempotencyKey: pgtype.Text{String: journeyID.String() + ":" + lastSampleID.String(), Valid: true},
			MaxAttempts:    8,
			RunAt:          pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("enqueue track processing: %w", err)
		}
		return nil
	})
	if err != nil {
		return IngestResult{}, err
	}

	return IngestResult{
		Received: len(points), Inserted: int(inserted), Duplicates: len(points) - int(inserted), Flagged: flagged,
	}, nil
}

func (s *Service) classify(point Point) (classifiedPoint, error) {
	if point.SampleID == uuid.Nil || point.CapturedAt.IsZero() ||
		point.Latitude < -90 || point.Latitude > 90 || point.Longitude < -180 || point.Longitude > 180 ||
		(point.AccuracyM != nil && *point.AccuracyM < 0) ||
		(point.SpeedMPS != nil && *point.SpeedMPS < 0) ||
		(point.Heading != nil && (*point.Heading < 0 || *point.Heading > 360)) ||
		point.CapturedAt.After(s.now().Add(5*time.Minute)) {
		return classifiedPoint{}, ErrInvalidBatch
	}

	classified := classifiedPoint{Point: point, IsAccepted: true}
	var reason string
	switch {
	case point.AccuracyM != nil && *point.AccuracyM > 100:
		reason = "poor_accuracy"
	case point.SpeedMPS != nil && *point.SpeedMPS > 80:
		reason = "implausible_speed"
	}
	if reason != "" {
		classified.IsAccepted = false
		classified.RejectionReason = &reason
	}
	return classified, nil
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

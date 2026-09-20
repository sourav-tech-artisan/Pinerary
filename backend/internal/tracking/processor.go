package tracking

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type Processor struct {
	pool *pgxpool.Pool
}

type processJob struct {
	JourneyID uuid.UUID `json:"journey_id"`
}

func NewProcessor(pool *pgxpool.Pool) *Processor {
	return &Processor{pool: pool}
}

func (p *Processor) HandleJob(ctx context.Context, payload []byte) error {
	var job processJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("decode track processing job: %w", err)
	}
	if job.JourneyID == uuid.Nil {
		return fmt.Errorf("decode track processing job: journey ID is required")
	}
	return p.Process(ctx, job.JourneyID)
}

func (p *Processor) Process(ctx context.Context, journeyID uuid.UUID) error {
	queries := dbgen.New(p.pool)
	rows, err := queries.ListLocationSamplesForProcessing(ctx, pgUUID(journeyID))
	if err != nil {
		return fmt.Errorf("list samples for processing: %w", err)
	}
	samples := make([]Sample, 0, len(rows))
	for _, row := range rows {
		accuracy := 0.0
		if row.AccuracyM.Valid {
			accuracy = float64(row.AccuracyM.Float32)
		}
		samples = append(samples, Sample{
			DatabaseID: row.ID, CapturedAt: row.CapturedAt.Time,
			Latitude: row.Latitude, Longitude: row.Longitude,
			AccuracyM: accuracy, Accepted: row.IsAccepted,
		})
	}
	segments, decisions := Clean(samples)

	return database.WithinTransaction(ctx, p.pool, func(tx *dbgen.Queries) error {
		for _, decision := range decisions {
			if err := tx.UpdateLocationSampleQuality(ctx, dbgen.UpdateLocationSampleQualityParams{
				ID: decision.DatabaseID, IsAccepted: decision.Accepted,
				RejectionReason: pgtype.Text{String: decision.Reason, Valid: true},
			}); err != nil {
				return fmt.Errorf("update sample quality: %w", err)
			}
		}
		if err := tx.DeleteRouteSegments(ctx, pgUUID(journeyID)); err != nil {
			return fmt.Errorf("replace route segments: %w", err)
		}
		for index, segment := range segments {
			_, err := tx.CreateRouteSegment(ctx, dbgen.CreateRouteSegmentParams{
				JourneyID: pgUUID(journeyID), SegmentNumber: int32(index + 1),
				StartedAt: pgtype.Timestamptz{Time: segment.StartedAt, Valid: true},
				EndedAt:   pgtype.Timestamptz{Time: segment.EndedAt, Valid: true},
				RawWkt:    routeWKT(segment.Points), DistanceM: segment.DistanceM,
				DurationS: int32(segment.EndedAt.Sub(segment.StartedAt).Seconds()),
			})
			if err != nil {
				return fmt.Errorf("create route segment: %w", err)
			}
		}
		jobPayload, _ := json.Marshal(processJob{JourneyID: journeyID})
		_, err := tx.EnqueueJob(ctx, dbgen.EnqueueJobParams{
			JobType: "track.match", Payload: jobPayload,
			IdempotencyKey: pgtype.Text{String: journeyID.String(), Valid: true},
			MaxAttempts:    8,
			RunAt:          pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("enqueue map matching: %w", err)
		}
		return nil
	})
}

func routeWKT(points []Sample) string {
	coordinates := make([]string, 0, len(points))
	for _, point := range points {
		coordinates = append(coordinates, fmt.Sprintf("%.7f %.7f", point.Longitude, point.Latitude))
	}
	return "SRID=4326;LINESTRING(" + strings.Join(coordinates, ",") + ")"
}

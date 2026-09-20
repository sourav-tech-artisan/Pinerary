package journeys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

const activeJourneyConstraint = "journeys_one_active_per_owner_idx"

type Store interface {
	CompleteJourney(context.Context, dbgen.CompleteJourneyParams) (dbgen.Journey, error)
	ConvertOutingToTrip(context.Context, dbgen.ConvertOutingToTripParams) (dbgen.Journey, error)
	CreateJourney(context.Context, dbgen.CreateJourneyParams) (dbgen.Journey, error)
	GetJourney(context.Context, dbgen.GetJourneyParams) (dbgen.Journey, error)
	ListJourneys(context.Context, dbgen.ListJourneysParams) ([]dbgen.Journey, error)
	UpdateJourneyLabel(context.Context, dbgen.UpdateJourneyLabelParams) (dbgen.Journey, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

type CreateInput struct {
	OwnerID         uuid.UUID
	ClientRequestID uuid.UUID
	Kind            Kind
	Label           string
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Journey, error) {
	if err := ValidateNew(input.Kind, input.Label); err != nil {
		return Journey{}, err
	}
	if input.ClientRequestID == uuid.Nil {
		return Journey{}, fmt.Errorf("client request ID is required")
	}

	row, err := s.store.CreateJourney(ctx, dbgen.CreateJourneyParams{
		OwnerID:         toPGUUID(input.OwnerID),
		ClientRequestID: toPGUUID(input.ClientRequestID),
		Kind:            string(input.Kind),
		Label:           strings.TrimSpace(input.Label),
		StartedAt:       pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == activeJourneyConstraint {
			return Journey{}, ErrActiveJourneyExists
		}
		return Journey{}, fmt.Errorf("create journey: %w", err)
	}
	return fromRow(row), nil
}

func (s *Service) Get(ctx context.Context, ownerID, journeyID uuid.UUID) (Journey, error) {
	row, err := s.store.GetJourney(ctx, dbgen.GetJourneyParams{ID: toPGUUID(journeyID), OwnerID: toPGUUID(ownerID)})
	if err != nil {
		return Journey{}, mapNotFound(err)
	}
	return fromRow(row), nil
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID, before *time.Time, pageSize int32) ([]Journey, error) {
	var cursor pgtype.Timestamptz
	if before != nil {
		cursor = pgtype.Timestamptz{Time: before.UTC(), Valid: true}
	}
	rows, err := s.store.ListJourneys(ctx, dbgen.ListJourneysParams{
		OwnerID:         toPGUUID(ownerID),
		BeforeStartedAt: cursor,
		PageSize:        pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list journeys: %w", err)
	}
	result := make([]Journey, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromRow(row))
	}
	return result, nil
}

func (s *Service) Rename(ctx context.Context, ownerID, journeyID uuid.UUID, version int32, label string) (Journey, error) {
	if err := ValidateNew(KindTrip, label); err != nil {
		return Journey{}, ErrInvalidLabel
	}
	row, err := s.store.UpdateJourneyLabel(ctx, dbgen.UpdateJourneyLabelParams{
		ID: toPGUUID(journeyID), OwnerID: toPGUUID(ownerID), Label: strings.TrimSpace(label), Version: version,
	})
	if err != nil {
		return Journey{}, mapConflict(err)
	}
	return fromRow(row), nil
}

func (s *Service) Complete(ctx context.Context, ownerID, journeyID uuid.UUID, version int32) (Journey, error) {
	row, err := s.store.CompleteJourney(ctx, dbgen.CompleteJourneyParams{
		ID: toPGUUID(journeyID), OwnerID: toPGUUID(ownerID),
		EndedAt: pgtype.Timestamptz{Time: s.now().UTC(), Valid: true}, Version: version,
	})
	if err != nil {
		return Journey{}, mapConflict(err)
	}
	return fromRow(row), nil
}

func (s *Service) ConvertToTrip(ctx context.Context, ownerID, journeyID uuid.UUID, version int32) (Journey, error) {
	row, err := s.store.ConvertOutingToTrip(ctx, dbgen.ConvertOutingToTripParams{
		ID: toPGUUID(journeyID), OwnerID: toPGUUID(ownerID), Version: version,
	})
	if err != nil {
		return Journey{}, mapConflict(err)
	}
	return fromRow(row), nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func mapConflict(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	return err
}

func fromRow(row dbgen.Journey) Journey {
	var endedAt *time.Time
	if row.EndedAt.Valid {
		value := row.EndedAt.Time
		endedAt = &value
	}
	return Journey{
		ID: uuid.UUID(row.ID.Bytes), OwnerID: uuid.UUID(row.OwnerID.Bytes), Kind: Kind(row.Kind),
		Label: row.Label, Status: Status(row.Status), StartedAt: row.StartedAt.Time,
		EndedAt: endedAt, Version: row.Version,
	}
}

func toPGUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

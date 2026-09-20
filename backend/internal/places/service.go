package places

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type Service struct {
	pool  *pgxpool.Pool
	store *dbgen.Queries
}

type SaveInput struct {
	OwnerID         uuid.UUID
	ClientRequestID uuid.UUID
	Name            string
	Notes           string
	Coordinates     Coordinates
}

type PinInput struct {
	OwnerID         uuid.UUID
	JourneyID       uuid.UUID
	ClientRequestID uuid.UUID
	Name            string
	Note            string
	Coordinates     Coordinates
	CapturedAt      time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, store: dbgen.New(pool)}
}

func (s *Service) Save(ctx context.Context, input SaveInput) (Place, error) {
	if err := Validate(input.Name, input.Coordinates); err != nil {
		return Place{}, err
	}
	if input.ClientRequestID == uuid.Nil {
		return Place{}, fmt.Errorf("client request ID is required")
	}
	row, err := s.store.CreatePlace(ctx, createPlaceParams(input))
	if err != nil {
		return Place{}, fmt.Errorf("create place: %w", err)
	}
	return placeFromCreate(row), nil
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID, limit int32) ([]Place, error) {
	rows, err := s.store.ListPlaces(ctx, dbgen.ListPlacesParams{OwnerID: pgUUID(ownerID), Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("list places: %w", err)
	}
	result := make([]Place, 0, len(rows))
	for _, row := range rows {
		result = append(result, Place{
			ID: uuid.UUID(row.ID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
			Name: row.Name, Notes: row.Notes,
			Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude},
			CreatedAt:   row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, ownerID, placeID uuid.UUID, name, notes string) (Place, error) {
	if err := Validate(name, Coordinates{}); err != nil && !errors.Is(err, ErrInvalidCoordinates) {
		return Place{}, err
	}
	row, err := s.store.UpdatePlace(ctx, dbgen.UpdatePlaceParams{
		ID: pgUUID(placeID), OwnerID: pgUUID(ownerID), Name: strings.TrimSpace(name), Notes: notes,
	})
	if err != nil {
		return Place{}, mapNotFound(err)
	}
	return Place{
		ID: uuid.UUID(row.ID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		Name: row.Name, Notes: row.Notes,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude},
		CreatedAt:   row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (s *Service) Delete(ctx context.Context, ownerID, placeID uuid.UUID) error {
	count, err := s.store.DeletePlace(ctx, dbgen.DeletePlaceParams{ID: pgUUID(placeID), OwnerID: pgUUID(ownerID)})
	if err != nil {
		return fmt.Errorf("delete place: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Pin(ctx context.Context, input PinInput) (Stop, error) {
	if err := Validate(input.Name, input.Coordinates); err != nil {
		return Stop{}, err
	}
	if input.ClientRequestID == uuid.Nil || input.CapturedAt.IsZero() {
		return Stop{}, fmt.Errorf("client request ID and captured time are required")
	}

	existing, err := s.store.GetJourneyStopByClientRequest(ctx, dbgen.GetJourneyStopByClientRequestParams{
		JourneyID: pgUUID(input.JourneyID), ClientRequestID: pgUUID(input.ClientRequestID), OwnerID: pgUUID(input.OwnerID),
	})
	if err == nil {
		return stopFromExisting(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Stop{}, fmt.Errorf("check existing stop: %w", err)
	}

	var result Stop
	err = database.WithinTransaction(ctx, s.pool, func(queries *dbgen.Queries) error {
		_, err := queries.LockActiveJourney(ctx, dbgen.LockActiveJourneyParams{
			ID: pgUUID(input.JourneyID), OwnerID: pgUUID(input.OwnerID),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrJourneyNotActive
		}
		if err != nil {
			return fmt.Errorf("lock journey: %w", err)
		}

		place, err := queries.CreatePlace(ctx, createPlaceParams(SaveInput{
			OwnerID: input.OwnerID, ClientRequestID: input.ClientRequestID,
			Name: input.Name, Notes: input.Note, Coordinates: input.Coordinates,
		}))
		if err != nil {
			return fmt.Errorf("create place for stop: %w", err)
		}
		sequence, err := queries.NextJourneyStopSequence(ctx, pgUUID(input.JourneyID))
		if err != nil {
			return fmt.Errorf("get next stop sequence: %w", err)
		}
		stop, err := queries.CreateJourneyStop(ctx, dbgen.CreateJourneyStopParams{
			JourneyID: pgUUID(input.JourneyID), ClientRequestID: pgUUID(input.ClientRequestID),
			SequenceNumber: sequence, CapturedAt: pgtype.Timestamptz{Time: input.CapturedAt.UTC(), Valid: true},
			DisplayName: strings.TrimSpace(input.Name), Note: input.Note,
			PlaceID: place.ID, OwnerID: pgUUID(input.OwnerID),
		})
		if err != nil {
			return fmt.Errorf("create journey stop: %w", err)
		}
		result = stopFromCreate(stop)
		return nil
	})
	if err != nil {
		return Stop{}, err
	}
	return result, nil
}

func (s *Service) ListStops(ctx context.Context, ownerID, journeyID uuid.UUID) ([]Stop, error) {
	rows, err := s.store.ListJourneyStops(ctx, dbgen.ListJourneyStopsParams{
		JourneyID: pgUUID(journeyID), OwnerID: pgUUID(ownerID),
	})
	if err != nil {
		return nil, fmt.Errorf("list journey stops: %w", err)
	}
	result := make([]Stop, 0, len(rows))
	for _, row := range rows {
		result = append(result, stopFromList(row))
	}
	return result, nil
}

func (s *Service) UpdateStop(ctx context.Context, ownerID, journeyID, stopID uuid.UUID, name, note string) (Stop, error) {
	if err := Validate(name, Coordinates{}); err != nil && !errors.Is(err, ErrInvalidCoordinates) {
		return Stop{}, err
	}
	row, err := s.store.UpdateJourneyStopMetadata(ctx, dbgen.UpdateJourneyStopMetadataParams{
		ID: pgUUID(stopID), JourneyID: pgUUID(journeyID), OwnerID: pgUUID(ownerID),
		DisplayName: strings.TrimSpace(name), Note: note,
	})
	if err != nil {
		return Stop{}, mapNotFound(err)
	}
	return stopFromUpdate(row), nil
}

func createPlaceParams(input SaveInput) dbgen.CreatePlaceParams {
	return dbgen.CreatePlaceParams{
		OwnerID: pgUUID(input.OwnerID), ClientRequestID: pgUUID(input.ClientRequestID),
		Name: strings.TrimSpace(input.Name), Notes: input.Notes,
		Longitude: input.Coordinates.Longitude, Latitude: input.Coordinates.Latitude,
	}
}

func placeFromCreate(row dbgen.CreatePlaceRow) Place {
	return Place{
		ID: uuid.UUID(row.ID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		Name: row.Name, Notes: row.Notes,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude},
		CreatedAt:   row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func stopFromCreate(row dbgen.CreateJourneyStopRow) Stop {
	return Stop{ID: uuid.UUID(row.ID.Bytes), JourneyID: uuid.UUID(row.JourneyID.Bytes),
		PlaceID: uuid.UUID(row.PlaceID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		SequenceNumber: row.SequenceNumber, CapturedAt: row.CapturedAt.Time,
		DisplayName: row.DisplayName, Note: row.Note,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude}}
}

func stopFromExisting(row dbgen.GetJourneyStopByClientRequestRow) Stop {
	return Stop{ID: uuid.UUID(row.ID.Bytes), JourneyID: uuid.UUID(row.JourneyID.Bytes),
		PlaceID: uuid.UUID(row.PlaceID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		SequenceNumber: row.SequenceNumber, CapturedAt: row.CapturedAt.Time,
		DisplayName: row.DisplayName, Note: row.Note,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude}}
}

func stopFromList(row dbgen.ListJourneyStopsRow) Stop {
	return Stop{ID: uuid.UUID(row.ID.Bytes), JourneyID: uuid.UUID(row.JourneyID.Bytes),
		PlaceID: uuid.UUID(row.PlaceID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		SequenceNumber: row.SequenceNumber, CapturedAt: row.CapturedAt.Time,
		DisplayName: row.DisplayName, Note: row.Note,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude}}
}

func stopFromUpdate(row dbgen.UpdateJourneyStopMetadataRow) Stop {
	return Stop{ID: uuid.UUID(row.ID.Bytes), JourneyID: uuid.UUID(row.JourneyID.Bytes),
		PlaceID: uuid.UUID(row.PlaceID.Bytes), ClientRequestID: uuid.UUID(row.ClientRequestID.Bytes),
		SequenceNumber: row.SequenceNumber, CapturedAt: row.CapturedAt.Time,
		DisplayName: row.DisplayName, Note: row.Note,
		Coordinates: Coordinates{Latitude: row.Latitude, Longitude: row.Longitude}}
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

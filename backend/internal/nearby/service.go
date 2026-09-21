package nearby

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
)

const (
	defaultRadiusM   = 100_000
	candidateFactor  = 5
	maxCandidateSize = 100
)

var (
	ErrInvalidQuery       = errors.New("nearby query is invalid")
	ErrRoutingUnavailable = errors.New("road routing is temporarily unavailable")
)

type Service struct {
	store  *dbgen.Queries
	router routing.MatrixRouter
}

type Estimate struct {
	DistanceM float64 `json:"distance_m"`
	DurationS int32   `json:"duration_s"`
}

type Result struct {
	PlaceID       uuid.UUID                  `json:"place_id"`
	Name          string                     `json:"name"`
	Latitude      float64                    `json:"latitude"`
	Longitude     float64                    `json:"longitude"`
	StraightLineM float64                    `json:"straight_line_m"`
	Routes        map[routing.Mode]*Estimate `json:"routes"`
}

func NewService(store *dbgen.Queries, router routing.MatrixRouter) *Service {
	return &Service{store: store, router: router}
}

func (s *Service) Find(
	ctx context.Context,
	ownerID uuid.UUID,
	origin routing.Coordinate,
	limit int32,
	sortMode routing.Mode,
) ([]Result, error) {
	if origin.Latitude < -90 || origin.Latitude > 90 || origin.Longitude < -180 || origin.Longitude > 180 ||
		limit < 1 || limit > 50 || !validMode(sortMode) {
		return nil, ErrInvalidQuery
	}
	candidateLimit := min(limit*candidateFactor, maxCandidateSize)
	candidates, err := s.store.NearbyPlaceCandidates(ctx, dbgen.NearbyPlaceCandidatesParams{
		Longitude: origin.Longitude, Latitude: origin.Latitude, OwnerID: pgUUID(ownerID),
		RadiusM: defaultRadiusM, CandidateLimit: candidateLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("find nearby candidates: %w", err)
	}
	destinations := make([]routing.Coordinate, 0, len(candidates))
	results := make([]Result, 0, len(candidates))
	for _, candidate := range candidates {
		destinations = append(destinations, routing.Coordinate{Latitude: candidate.Latitude, Longitude: candidate.Longitude})
		results = append(results, Result{
			PlaceID: uuid.UUID(candidate.ID.Bytes), Name: candidate.Name,
			Latitude: candidate.Latitude, Longitude: candidate.Longitude,
			StraightLineM: candidate.StraightLineM, Routes: make(map[routing.Mode]*Estimate),
		})
	}

	for _, mode := range []routing.Mode{routing.ModeMotorcycle, routing.ModeCar, routing.ModeWalking} {
		costs, matrixErr := s.router.Matrix(ctx, origin, destinations, mode)
		if matrixErr != nil {
			if mode == sortMode {
				return nil, fmt.Errorf("%w: %v", ErrRoutingUnavailable, matrixErr)
			}
			continue
		}
		for index, cost := range costs {
			if !cost.Reachable {
				continue
			}
			results[index].Routes[mode] = &Estimate{DistanceM: cost.DistanceM, DurationS: cost.Duration}
		}
	}
	reachable := results[:0]
	for _, result := range results {
		if result.Routes[sortMode] != nil {
			reachable = append(reachable, result)
		}
	}
	results = reachable

	sort.SliceStable(results, func(left, right int) bool {
		return results[left].Routes[sortMode].DistanceM < results[right].Routes[sortMode].DistanceM
	})
	if len(results) > int(limit) {
		results = results[:limit]
	}
	return results, nil
}

func validMode(mode routing.Mode) bool {
	return mode == routing.ModeMotorcycle || mode == routing.ModeCar || mode == routing.ModeWalking
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

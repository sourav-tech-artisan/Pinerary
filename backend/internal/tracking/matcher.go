package tracking

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
)

type Matcher struct {
	store    *dbgen.Queries
	provider routing.MapMatcher
}

func NewMatcher(store *dbgen.Queries, provider routing.MapMatcher) *Matcher {
	return &Matcher{store: store, provider: provider}
}

func (m *Matcher) HandleJob(ctx context.Context, payload []byte) error {
	var job processJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("decode map matching job: %w", err)
	}
	if job.JourneyID == uuid.Nil {
		return fmt.Errorf("decode map matching job: journey ID is required")
	}

	segments, err := m.store.ListRouteSegmentsForMatching(ctx, pgUUID(job.JourneyID))
	if err != nil {
		return fmt.Errorf("list segments for matching: %w", err)
	}
	for _, segment := range segments {
		var geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		}
		if err := json.Unmarshal([]byte(segment.Geojson), &geometry); err != nil {
			return fmt.Errorf("decode raw segment: %w", err)
		}
		input := make([]routing.Coordinate, 0, len(geometry.Coordinates))
		for _, coordinate := range geometry.Coordinates {
			if len(coordinate) < 2 {
				return fmt.Errorf("raw segment contains an invalid coordinate")
			}
			input = append(input, routing.Coordinate{Latitude: coordinate[1], Longitude: coordinate[0]})
		}
		matched, err := m.provider.Match(ctx, input)
		if err != nil {
			return err
		}
		if err := m.store.UpdateMatchedRouteSegment(ctx, dbgen.UpdateMatchedRouteSegmentParams{
			ID: segment.ID, MatchedWkt: matchedWKT(matched),
		}); err != nil {
			return fmt.Errorf("save matched segment: %w", err)
		}
	}
	return nil
}

func matchedWKT(coordinates []routing.Coordinate) string {
	parts := make([]string, 0, len(coordinates))
	for _, coordinate := range coordinates {
		parts = append(parts, fmt.Sprintf("%.7f %.7f", coordinate.Longitude, coordinate.Latitude))
	}
	return "SRID=4326;LINESTRING(" + strings.Join(parts, ",") + ")"
}

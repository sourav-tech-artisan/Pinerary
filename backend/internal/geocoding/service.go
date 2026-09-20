package geocoding

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type Cache interface {
	GetReverseGeocodeCache(context.Context, dbgen.GetReverseGeocodeCacheParams) (dbgen.ReverseGeocodeCache, error)
	UpsertReverseGeocodeCache(context.Context, dbgen.UpsertReverseGeocodeCacheParams) (dbgen.ReverseGeocodeCache, error)
}

type Service struct {
	cache    Cache
	provider ReverseGeocoder
}

func NewService(cache Cache, provider ReverseGeocoder) *Service {
	return &Service{cache: cache, provider: provider}
}

func (s *Service) Reverse(ctx context.Context, latitude, longitude float64) (Result, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return Result{}, fmt.Errorf("coordinates are invalid")
	}
	key := dbgen.GetReverseGeocodeCacheParams{
		LatitudeE5:  int32(math.Round(latitude * 100000)),
		LongitudeE5: int32(math.Round(longitude * 100000)),
	}
	cached, err := s.cache.GetReverseGeocodeCache(ctx, key)
	if err == nil {
		return Result{DisplayName: cached.DisplayName, Raw: cached.ProviderPayload}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, fmt.Errorf("read geocoding cache: %w", err)
	}

	result, err := s.provider.Reverse(ctx, latitude, longitude)
	if err != nil {
		return Result{}, err
	}
	_, err = s.cache.UpsertReverseGeocodeCache(ctx, dbgen.UpsertReverseGeocodeCacheParams{
		LatitudeE5: key.LatitudeE5, LongitudeE5: key.LongitudeE5,
		DisplayName: result.DisplayName, ProviderPayload: result.Raw,
	})
	if err != nil {
		return Result{}, fmt.Errorf("write geocoding cache: %w", err)
	}
	return result, nil
}

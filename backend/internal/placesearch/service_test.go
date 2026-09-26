package placesearch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

type searchStoreStub struct {
	cache        dbgen.PlaceSearchCache
	cacheErr     error
	reserveErr   error
	reserved     dbgen.ReserveProviderDailyUsageParams
	upserted     dbgen.UpsertPlaceSearchCacheParams
	reserveCalls int
	upsertCalls  int
}

func (s *searchStoreStub) GetPlaceSearchCache(context.Context, string) (dbgen.PlaceSearchCache, error) {
	return s.cache, s.cacheErr
}

func (s *searchStoreStub) ReserveProviderDailyUsage(_ context.Context, params dbgen.ReserveProviderDailyUsageParams) (dbgen.ProviderDailyUsage, error) {
	s.reserveCalls++
	s.reserved = params
	return dbgen.ProviderDailyUsage{}, s.reserveErr
}

func (s *searchStoreStub) UpsertPlaceSearchCache(_ context.Context, params dbgen.UpsertPlaceSearchCacheParams) (dbgen.PlaceSearchCache, error) {
	s.upsertCalls++
	s.upserted = params
	return dbgen.PlaceSearchCache{}, nil
}

type searchProviderStub struct {
	results []Suggestion
	err     error
	input   SuggestInput
	calls   int
}

func (p *searchProviderStub) Name() string { return ProviderGeoapify }

func (p *searchProviderStub) Suggest(_ context.Context, input SuggestInput) ([]Suggestion, error) {
	p.calls++
	p.input = input
	return p.results, p.err
}

func TestSearchReturnsCachedSuggestions(t *testing.T) {
	t.Parallel()

	want := []Suggestion{{ResultID: "cached", Name: "Cached place", Latitude: 28.6, Longitude: 77.2}}
	encoded, _ := json.Marshal(want)
	store := &searchStoreStub{cache: dbgen.PlaceSearchCache{Results: encoded}}
	provider := &searchProviderStub{}
	service := NewService(store, provider, 2500)

	got, err := service.Search(context.Background(), SuggestInput{Query: "Cached", Limit: 5})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].ResultID != want[0].ResultID {
		t.Fatalf("unexpected cached result %#v", got)
	}
	if provider.calls != 0 || store.reserveCalls != 0 {
		t.Fatal("cache hit called provider or consumed budget")
	}
}

func TestSearchNormalizesAndCachesProviderSuggestions(t *testing.T) {
	t.Parallel()

	store := &searchStoreStub{cacheErr: pgx.ErrNoRows}
	provider := &searchProviderStub{results: []Suggestion{{
		ResultID: "result", Name: "Bhappe Da Hotel", Address: "New Delhi", Latitude: 28.6, Longitude: 77.2,
	}}}
	service := NewService(store, provider, 2500)
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	got, err := service.Search(context.Background(), SuggestInput{
		Query: "  Bhappe   Da  ", Limit: 5, AcceptLanguage: "en-IN,en;q=0.9",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || provider.input.Query != "Bhappe Da" || provider.input.AcceptLanguage != "en" {
		t.Fatalf("unexpected provider input/result: input=%#v result=%#v", provider.input, got)
	}
	if store.reserved.Provider != ProviderGeoapify || store.reserved.DailyLimit != 2500 {
		t.Fatalf("unexpected budget reservation %#v", store.reserved)
	}
	if len(store.upserted.CacheKey) != 64 || store.upserted.ResultCount != 1 {
		t.Fatalf("unexpected cache write %#v", store.upserted)
	}
	if !store.upserted.ExpiresAt.Time.Equal(now.Add(defaultResultTTL)) {
		t.Fatalf("unexpected cache expiry %v", store.upserted.ExpiresAt.Time)
	}
}

func TestSearchUsesShortTTLForEmptyResults(t *testing.T) {
	t.Parallel()

	store := &searchStoreStub{cacheErr: pgx.ErrNoRows}
	service := NewService(store, &searchProviderStub{results: nil}, 2500)
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	got, err := service.Search(context.Background(), SuggestInput{Query: "Unknown", Limit: 5})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty result, got %#v", got)
	}
	if !store.upserted.ExpiresAt.Time.Equal(now.Add(emptyResultTTL)) {
		t.Fatalf("unexpected empty-result expiry %v", store.upserted.ExpiresAt.Time)
	}
}

func TestSearchStopsWhenDailyBudgetIsExhausted(t *testing.T) {
	t.Parallel()

	store := &searchStoreStub{cacheErr: pgx.ErrNoRows, reserveErr: pgx.ErrNoRows}
	provider := &searchProviderStub{}
	service := NewService(store, provider, 2500)

	_, err := service.Search(context.Background(), SuggestInput{Query: "Delhi", Limit: 5})
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if provider.calls != 0 || store.upsertCalls != 0 {
		t.Fatal("quota exhaustion still called provider or wrote cache")
	}
}

func TestSearchValidatesInputBeforeUsingStore(t *testing.T) {
	t.Parallel()

	service := NewService(&searchStoreStub{cacheErr: errors.New("must not be called")}, &searchProviderStub{}, 2500)
	_, err := service.Search(context.Background(), SuggestInput{Query: "ab", Limit: 11})
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected invalid query, got %v", err)
	}
}

func TestSecondsUntilUTCReset(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 27, 23, 59, 30, 0, time.FixedZone("IST", 5*60*60+30*60))
	if got := SecondsUntilUTCReset(now); got != 19830 {
		t.Fatalf("expected 19830 seconds until UTC reset, got %d", got)
	}
}

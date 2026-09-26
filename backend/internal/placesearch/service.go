package placesearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

const (
	defaultResultTTL = 7 * 24 * time.Hour
	emptyResultTTL   = time.Hour
	minQueryRunes    = 3
	maxQueryRunes    = 200
	maxResultLimit   = 10
)

var (
	ErrInvalidQuery  = errors.New("place search query is invalid")
	ErrQuotaExceeded = errors.New("place search daily allowance is exhausted")
)

type Store interface {
	GetPlaceSearchCache(context.Context, string) (dbgen.PlaceSearchCache, error)
	ReserveProviderDailyUsage(context.Context, dbgen.ReserveProviderDailyUsageParams) (dbgen.ProviderDailyUsage, error)
	UpsertPlaceSearchCache(context.Context, dbgen.UpsertPlaceSearchCacheParams) (dbgen.PlaceSearchCache, error)
}

type Service struct {
	store       Store
	provider    Provider
	dailyBudget int32
	now         func() time.Time
}

func NewService(store Store, provider Provider, dailyBudget int32) *Service {
	return &Service{store: store, provider: provider, dailyBudget: dailyBudget, now: time.Now}
}

func (s *Service) Search(ctx context.Context, input SuggestInput) ([]Suggestion, error) {
	input.Query = normalizeQuery(input.Query)
	input.AcceptLanguage = preferredLanguage(input.AcceptLanguage)
	if s.provider == nil || s.store == nil {
		return nil, ErrProviderUnavailable
	}
	if utf8.RuneCountInString(input.Query) < minQueryRunes || utf8.RuneCountInString(input.Query) > maxQueryRunes ||
		input.Limit < 1 || input.Limit > maxResultLimit {
		return nil, ErrInvalidQuery
	}

	cacheKey := searchCacheKey(s.provider.Name(), input)
	cached, err := s.store.GetPlaceSearchCache(ctx, cacheKey)
	if err == nil {
		var suggestions []Suggestion
		if err := json.Unmarshal(cached.Results, &suggestions); err != nil {
			return nil, fmt.Errorf("decode place search cache: %w", err)
		}
		if suggestions == nil {
			suggestions = []Suggestion{}
		}
		return suggestions, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read place search cache: %w", err)
	}

	_, err = s.store.ReserveProviderDailyUsage(ctx, dbgen.ReserveProviderDailyUsageParams{
		Provider: s.provider.Name(), DailyLimit: s.dailyBudget,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuotaExceeded
	}
	if err != nil {
		return nil, fmt.Errorf("reserve place search provider usage: %w", err)
	}

	suggestions, err := s.provider.Suggest(ctx, input)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	if suggestions == nil {
		suggestions = []Suggestion{}
	}
	encoded, err := json.Marshal(suggestions)
	if err != nil {
		return nil, fmt.Errorf("encode place search cache: %w", err)
	}
	ttl := defaultResultTTL
	if len(suggestions) == 0 {
		ttl = emptyResultTTL
	}
	_, err = s.store.UpsertPlaceSearchCache(ctx, dbgen.UpsertPlaceSearchCacheParams{
		CacheKey: cacheKey, Results: encoded, ResultCount: int32(len(suggestions)),
		ExpiresAt: pgtype.Timestamptz{Time: s.now().UTC().Add(ttl), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("write place search cache: %w", err)
	}
	return suggestions, nil
}

func normalizeQuery(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func preferredLanguage(value string) string {
	first, _, _ := strings.Cut(value, ",")
	first, _, _ = strings.Cut(first, ";")
	first, _, _ = strings.Cut(first, "-")
	first = strings.ToLower(strings.TrimSpace(first))
	if len(first) != 2 {
		return ""
	}
	for _, character := range first {
		if !unicode.IsLetter(character) || character > unicode.MaxASCII {
			return ""
		}
	}
	return first
}

func searchCacheKey(provider string, input SuggestInput) string {
	value := strings.Join([]string{
		provider,
		strings.ToLower(input.Query),
		input.AcceptLanguage,
		fmt.Sprintf("%d", input.Limit),
	}, "\x00")
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func SecondsUntilUTCReset(now time.Time) int {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	seconds := int(next.Sub(now).Seconds())
	return max(1, seconds)
}

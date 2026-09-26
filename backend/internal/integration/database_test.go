//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
)

func TestPlaceSearchCacheAndDailyBudget(t *testing.T) {
	pool := openTestPool(t)
	queries := dbgen.New(pool)
	ctx := context.Background()
	cacheKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	provider := "integration-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM place_search_cache WHERE cache_key = $1", cacheKey)
		_, _ = pool.Exec(context.Background(), "DELETE FROM provider_daily_usage WHERE provider = $1", provider)
	})

	_, err := queries.UpsertPlaceSearchCache(ctx, dbgen.UpsertPlaceSearchCacheParams{
		CacheKey: cacheKey, Results: []byte(`[{"name":"Delhi"}]`), ResultCount: 1,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("upsert place search cache: %v", err)
	}
	cache, err := queries.GetPlaceSearchCache(ctx, cacheKey)
	if err != nil || cache.ResultCount != 1 {
		t.Fatalf("get place search cache: cache=%#v err=%v", cache, err)
	}

	for expected := int32(1); expected <= 2; expected++ {
		usage, err := queries.ReserveProviderDailyUsage(ctx, dbgen.ReserveProviderDailyUsageParams{
			Provider: provider, DailyLimit: 2,
		})
		if err != nil || usage.RequestCount != expected {
			t.Fatalf("reserve provider usage %d: usage=%#v err=%v", expected, usage, err)
		}
	}
	_, err = queries.ReserveProviderDailyUsage(ctx, dbgen.ReserveProviderDailyUsageParams{
		Provider: provider, DailyLimit: 2,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected exhausted budget to return no rows, got %v", err)
	}
}

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("PINERARY_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PINERARY_DATABASE_URL is required for integration tests")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestOwnershipIsolationAndSpatialCandidates(t *testing.T) {
	pool := openTestPool(t)
	queries := dbgen.New(pool)
	ctx := context.Background()
	owner := createUser(t, ctx, pool, queries, "integration-owner-"+uuid.NewString())
	other := createUser(t, ctx, pool, queries, "integration-other-"+uuid.NewString())

	created, err := queries.CreatePlace(ctx, dbgen.CreatePlaceParams{
		OwnerID: owner.ID, ClientRequestID: pgUUID(uuid.New()), Name: "Integration Fort",
		Longitude: 75.8155, Latitude: 26.9373,
	})
	if err != nil {
		t.Fatalf("create place: %v", err)
	}

	ownerResults, err := queries.NearbyPlaceCandidates(ctx, dbgen.NearbyPlaceCandidatesParams{
		OwnerID: owner.ID, Latitude: 26.94, Longitude: 75.82, RadiusM: 10_000, CandidateLimit: 10,
	})
	if err != nil {
		t.Fatalf("query owner candidates: %v", err)
	}
	if len(ownerResults) != 1 || ownerResults[0].ID != created.ID {
		t.Fatalf("owner candidates = %#v", ownerResults)
	}

	otherResults, err := queries.NearbyPlaceCandidates(ctx, dbgen.NearbyPlaceCandidatesParams{
		OwnerID: other.ID, Latitude: 26.94, Longitude: 75.82, RadiusM: 10_000, CandidateLimit: 10,
	})
	if err != nil {
		t.Fatalf("query other user's candidates: %v", err)
	}
	if len(otherResults) != 0 {
		t.Fatalf("other user saw private places: %#v", otherResults)
	}
}

func TestJourneyCreateIsIdempotent(t *testing.T) {
	pool := openTestPool(t)
	queries := dbgen.New(pool)
	ctx := context.Background()
	owner := createUser(t, ctx, pool, queries, "integration-journey-"+uuid.NewString())
	requestID := pgUUID(uuid.New())
	params := dbgen.CreateJourneyParams{
		OwnerID: owner.ID, ClientRequestID: requestID, Kind: "trip", Label: "Integration trip",
		StartedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	first, err := queries.CreateJourney(ctx, params)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := queries.CreateJourney(ctx, params)
	if err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent create returned IDs %v and %v", first.ID, second.ID)
	}
}

func createUser(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *dbgen.Queries,
	subject string,
) dbgen.User {
	t.Helper()
	user, err := queries.UpsertUserFromIdentity(ctx, dbgen.UpsertUserFromIdentityParams{
		OidcSubject: subject, DisplayName: "Integration User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})
	return user
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

-- name: GetReverseGeocodeCache :one
SELECT * FROM reverse_geocode_cache
WHERE latitude_e5 = $1 AND longitude_e5 = $2;

-- name: UpsertReverseGeocodeCache :one
INSERT INTO reverse_geocode_cache (latitude_e5, longitude_e5, display_name, provider_payload)
VALUES ($1, $2, $3, $4)
ON CONFLICT (latitude_e5, longitude_e5) DO UPDATE
SET display_name = EXCLUDED.display_name,
    provider_payload = EXCLUDED.provider_payload,
    updated_at = now()
RETURNING *;

-- name: GetPlaceSearchCache :one
SELECT * FROM place_search_cache
WHERE cache_key = $1
  AND expires_at > now();

-- name: UpsertPlaceSearchCache :one
INSERT INTO place_search_cache (cache_key, results, result_count, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (cache_key) DO UPDATE
SET results = EXCLUDED.results,
    result_count = EXCLUDED.result_count,
    expires_at = EXCLUDED.expires_at,
    updated_at = now()
RETURNING *;

-- name: ReserveProviderDailyUsage :one
INSERT INTO provider_daily_usage (provider, usage_date, request_count)
VALUES (sqlc.arg(provider), (now() AT TIME ZONE 'UTC')::date, 1)
ON CONFLICT (provider, usage_date) DO UPDATE
SET request_count = provider_daily_usage.request_count + 1,
    updated_at = now()
WHERE provider_daily_usage.request_count < sqlc.arg(daily_limit)
RETURNING *;

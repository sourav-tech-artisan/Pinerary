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

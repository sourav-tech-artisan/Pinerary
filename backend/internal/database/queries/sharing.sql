-- name: CreateItineraryShare :one
INSERT INTO itinerary_shares (journey_id, owner_id, token_hash, snapshot, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetItineraryShareByTokenHash :one
SELECT * FROM itinerary_shares
WHERE token_hash = $1 AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now());

-- name: RevokeItineraryShare :execrows
UPDATE itinerary_shares SET revoked_at = now()
WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL;

-- name: ListPhotosForShare :many
SELECT id, stop_id, object_key, thumbnail_key
FROM photos
WHERE owner_id = $1 AND journey_id = $2 AND status = 'processed';

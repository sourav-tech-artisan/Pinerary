-- name: CreatePhotoUpload :one
INSERT INTO photos (
    id, owner_id, journey_id, stop_id, client_request_id, object_key,
    content_type, status, captured_at
)
SELECT sqlc.arg(id), sqlc.arg(owner_id), journeys.id, sqlc.narg(stop_id),
    sqlc.arg(client_request_id), sqlc.arg(object_key), sqlc.arg(content_type),
    'pending', sqlc.narg(captured_at)
FROM journeys
WHERE journeys.id = sqlc.arg(journey_id) AND journeys.owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(stop_id)::uuid IS NULL OR EXISTS (
      SELECT 1 FROM journey_stops
      WHERE journey_stops.id = sqlc.narg(stop_id) AND journey_stops.journey_id = journeys.id
  ))
ON CONFLICT (owner_id, client_request_id) WHERE client_request_id IS NOT NULL
DO UPDATE SET client_request_id = photos.client_request_id
RETURNING *;

-- name: GetPhotoForOwner :one
SELECT * FROM photos WHERE id = $1 AND owner_id = $2;

-- name: CompletePhotoUpload :one
UPDATE photos
SET status = 'uploaded', byte_size = $3, checksum_sha256 = $4, updated_at = now()
WHERE id = $1 AND owner_id = $2 AND status IN ('pending', 'uploaded')
RETURNING *;

-- name: MarkPhotoProcessed :one
UPDATE photos
SET status = 'processed', thumbnail_key = $2, updated_at = now()
WHERE id = $1 AND status IN ('uploaded', 'processed')
RETURNING *;

-- name: MarkPhotoFailed :exec
UPDATE photos SET status = 'failed', updated_at = now() WHERE id = $1;

-- name: ListStopPhotos :many
SELECT * FROM photos
WHERE owner_id = $1 AND stop_id = $2 AND status = 'processed'
ORDER BY captured_at NULLS LAST, created_at;

-- name: CreateJourney :one
INSERT INTO journeys (owner_id, client_request_id, kind, label, started_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (owner_id, client_request_id) DO UPDATE
SET client_request_id = journeys.client_request_id
RETURNING *;

-- name: GetJourney :one
SELECT * FROM journeys WHERE id = $1 AND owner_id = $2;

-- name: ListJourneys :many
SELECT * FROM journeys
WHERE owner_id = $1
  AND (sqlc.narg(before_started_at)::timestamptz IS NULL OR started_at < sqlc.narg(before_started_at))
ORDER BY started_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- name: UpdateJourneyLabel :one
UPDATE journeys
SET label = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND owner_id = $2 AND version = $4
RETURNING *;

-- name: CompleteJourney :one
UPDATE journeys
SET status = 'ended', ended_at = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND owner_id = $2 AND status = 'active' AND version = $4
RETURNING *;

-- name: ExpireJourney :one
UPDATE journeys
SET status = 'expired', ended_at = $2, version = version + 1, updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: ConvertOutingToTrip :one
UPDATE journeys
SET kind = 'trip', version = version + 1, updated_at = now()
WHERE id = $1 AND owner_id = $2 AND kind = 'outing' AND status = 'active' AND version = $3
RETURNING *;

-- name: ListActiveOutingsBefore :many
SELECT * FROM journeys
WHERE kind = 'outing' AND status = 'active' AND started_at <= $1
ORDER BY started_at
LIMIT $2;

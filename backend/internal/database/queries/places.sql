-- name: CreatePlace :one
INSERT INTO places (owner_id, client_request_id, name, notes, location)
VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint(
    sqlc.arg(longitude)::double precision,
    sqlc.arg(latitude)::double precision
), 4326)::geography)
ON CONFLICT (owner_id, client_request_id) WHERE client_request_id IS NOT NULL
DO UPDATE SET client_request_id = places.client_request_id
RETURNING id, owner_id, client_request_id, name, notes,
    ST_Y(location::geometry)::double precision AS latitude,
    ST_X(location::geometry)::double precision AS longitude,
    created_at, updated_at;

-- name: GetPlace :one
SELECT id, owner_id, client_request_id, name, notes,
    ST_Y(location::geometry)::double precision AS latitude,
    ST_X(location::geometry)::double precision AS longitude,
    created_at, updated_at
FROM places
WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL;

-- name: ListPlaces :many
SELECT id, owner_id, client_request_id, name, notes,
    ST_Y(location::geometry)::double precision AS latitude,
    ST_X(location::geometry)::double precision AS longitude,
    created_at, updated_at
FROM places
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: UpdatePlace :one
UPDATE places
SET name = $3, notes = $4, updated_at = now()
WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
RETURNING id, owner_id, client_request_id, name, notes,
    ST_Y(location::geometry)::double precision AS latitude,
    ST_X(location::geometry)::double precision AS longitude,
    created_at, updated_at;

-- name: DeletePlace :execrows
UPDATE places SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL;

-- name: LockActiveJourney :one
SELECT id FROM journeys
WHERE id = $1 AND owner_id = $2 AND status = 'active'
FOR UPDATE;

-- name: NextJourneyStopSequence :one
SELECT (coalesce(max(sequence_number), 0) + 1)::bigint
FROM journey_stops WHERE journey_id = $1;

-- name: CreateJourneyStop :one
INSERT INTO journey_stops (
    journey_id, place_id, client_request_id, sequence_number, captured_at,
    display_name, note, location
)
SELECT sqlc.arg(journey_id), places.id, sqlc.arg(client_request_id), sqlc.arg(sequence_number),
    sqlc.arg(captured_at), sqlc.arg(display_name), sqlc.arg(note), places.location
FROM places
WHERE places.id = sqlc.arg(place_id) AND places.owner_id = sqlc.arg(owner_id) AND places.deleted_at IS NULL
ON CONFLICT (journey_id, client_request_id) WHERE client_request_id IS NOT NULL
DO UPDATE SET client_request_id = journey_stops.client_request_id
RETURNING id, journey_id, place_id, client_request_id, sequence_number, captured_at,
    display_name, note,
    ST_Y(location::geometry)::double precision AS latitude,
    ST_X(location::geometry)::double precision AS longitude,
    created_at, updated_at;

-- name: ListJourneyStops :many
SELECT stops.id, stops.journey_id, stops.place_id, stops.client_request_id,
    stops.sequence_number, stops.captured_at, stops.display_name, stops.note,
    ST_Y(stops.location::geometry)::double precision AS latitude,
    ST_X(stops.location::geometry)::double precision AS longitude,
    stops.created_at, stops.updated_at
FROM journey_stops AS stops
JOIN journeys ON journeys.id = stops.journey_id
WHERE stops.journey_id = $1 AND journeys.owner_id = $2
ORDER BY stops.sequence_number;

-- name: GetJourneyStopByClientRequest :one
SELECT stops.id, stops.journey_id, stops.place_id, stops.client_request_id,
    stops.sequence_number, stops.captured_at, stops.display_name, stops.note,
    ST_Y(stops.location::geometry)::double precision AS latitude,
    ST_X(stops.location::geometry)::double precision AS longitude,
    stops.created_at, stops.updated_at
FROM journey_stops AS stops
JOIN journeys ON journeys.id = stops.journey_id
WHERE stops.journey_id = $1 AND stops.client_request_id = $2 AND journeys.owner_id = $3;

-- name: UpdateJourneyStopMetadata :one
UPDATE journey_stops AS stops
SET display_name = $4, note = $5, updated_at = now()
FROM journeys
WHERE stops.id = $1 AND stops.journey_id = $2
  AND journeys.id = stops.journey_id AND journeys.owner_id = $3
RETURNING stops.id, stops.journey_id, stops.place_id, stops.client_request_id,
    stops.sequence_number, stops.captured_at, stops.display_name, stops.note,
    ST_Y(stops.location::geometry)::double precision AS latitude,
    ST_X(stops.location::geometry)::double precision AS longitude,
    stops.created_at, stops.updated_at;

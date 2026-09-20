-- name: InsertLocationSamples :execrows
INSERT INTO location_samples (
    journey_id, user_id, sample_id, captured_at, location,
    accuracy_m, speed_mps, heading_deg, is_accepted, rejection_reason
)
SELECT sqlc.arg(journey_id), sqlc.arg(user_id), points.sample_id, points.captured_at,
    ST_SetSRID(ST_MakePoint(points.longitude, points.latitude), 4326)::geography,
    points.accuracy_m, points.speed_mps, points.heading_deg,
    points.is_accepted, points.rejection_reason
FROM jsonb_to_recordset(sqlc.arg(points)::jsonb) AS points(
    sample_id uuid,
    captured_at timestamptz,
    latitude double precision,
    longitude double precision,
    accuracy_m real,
    speed_mps real,
    heading_deg real,
    is_accepted boolean,
    rejection_reason text
)
ON CONFLICT (journey_id, sample_id) DO NOTHING;

-- name: ListLocationSamples :many
SELECT samples.sample_id, samples.captured_at,
    ST_Y(samples.location::geometry)::double precision AS latitude,
    ST_X(samples.location::geometry)::double precision AS longitude,
    samples.accuracy_m, samples.speed_mps, samples.heading_deg,
    samples.is_accepted, samples.rejection_reason
FROM location_samples AS samples
JOIN journeys ON journeys.id = samples.journey_id
WHERE samples.journey_id = $1 AND journeys.owner_id = $2
ORDER BY samples.captured_at, samples.id;

-- name: ListLocationSamplesForProcessing :many
SELECT samples.id, samples.sample_id, samples.captured_at,
    ST_Y(samples.location::geometry)::double precision AS latitude,
    ST_X(samples.location::geometry)::double precision AS longitude,
    samples.accuracy_m, samples.speed_mps, samples.heading_deg,
    samples.is_accepted, samples.rejection_reason
FROM location_samples AS samples
WHERE samples.journey_id = $1
ORDER BY samples.captured_at, samples.id;

-- name: UpdateLocationSampleQuality :exec
UPDATE location_samples
SET is_accepted = $2, rejection_reason = $3
WHERE id = $1;

-- name: DeleteRouteSegments :exec
DELETE FROM route_segments WHERE journey_id = $1;

-- name: CreateRouteSegment :one
INSERT INTO route_segments (
    journey_id, segment_number, started_at, ended_at, raw_path, distance_m, duration_s
)
VALUES (
    $1, $2, $3, $4, ST_GeogFromText(sqlc.arg(raw_wkt)), $5, $6
)
RETURNING id;

-- name: GetJourneyRoute :many
SELECT segments.id, segments.segment_number, segments.started_at, segments.ended_at,
    ST_AsGeoJSON(coalesce(segments.matched_path, segments.raw_path)::geometry)::text AS geojson,
    segments.distance_m, segments.duration_s,
    (segments.matched_path IS NOT NULL)::boolean AS is_map_matched
FROM route_segments AS segments
JOIN journeys ON journeys.id = segments.journey_id
WHERE segments.journey_id = $1 AND journeys.owner_id = $2
ORDER BY segments.segment_number;

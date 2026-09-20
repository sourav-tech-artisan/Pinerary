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

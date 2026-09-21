-- name: NearbyPlaceCandidates :many
WITH origin AS (
    SELECT ST_SetSRID(ST_MakePoint(
        sqlc.arg(longitude)::double precision,
        sqlc.arg(latitude)::double precision
    ), 4326)::geography AS location
)
SELECT places.id, places.name,
    ST_Y(places.location::geometry)::double precision AS latitude,
    ST_X(places.location::geometry)::double precision AS longitude,
    ST_Distance(places.location, origin.location)::double precision AS straight_line_m
FROM places
CROSS JOIN origin
WHERE places.owner_id = sqlc.arg(owner_id)
  AND places.deleted_at IS NULL
  AND ST_DWithin(places.location, origin.location, sqlc.arg(radius_m)::double precision)
ORDER BY ST_Distance(places.location, origin.location), places.id
LIMIT sqlc.arg(candidate_limit);

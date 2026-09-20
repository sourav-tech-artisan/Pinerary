-- +goose Up
ALTER TABLE places ADD COLUMN client_request_id uuid;
ALTER TABLE journey_stops ADD COLUMN client_request_id uuid;

CREATE UNIQUE INDEX places_owner_client_request_idx
    ON places (owner_id, client_request_id) WHERE client_request_id IS NOT NULL;
CREATE UNIQUE INDEX journey_stops_client_request_idx
    ON journey_stops (journey_id, client_request_id) WHERE client_request_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS journey_stops_client_request_idx;
DROP INDEX IF EXISTS places_owner_client_request_idx;
ALTER TABLE journey_stops DROP COLUMN IF EXISTS client_request_id;
ALTER TABLE places DROP COLUMN IF EXISTS client_request_id;

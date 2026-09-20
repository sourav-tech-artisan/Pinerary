-- +goose Up
ALTER TABLE photos ADD COLUMN client_request_id uuid;
ALTER TABLE photos ADD COLUMN checksum_sha256 text;

CREATE UNIQUE INDEX photos_owner_client_request_idx
    ON photos (owner_id, client_request_id) WHERE client_request_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS photos_owner_client_request_idx;
ALTER TABLE photos DROP COLUMN IF EXISTS checksum_sha256;
ALTER TABLE photos DROP COLUMN IF EXISTS client_request_id;

-- +goose Up
CREATE TABLE reverse_geocode_cache (
    latitude_e5 integer NOT NULL,
    longitude_e5 integer NOT NULL,
    display_name text NOT NULL,
    provider_payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (latitude_e5, longitude_e5)
);

-- +goose Down
DROP TABLE IF EXISTS reverse_geocode_cache;

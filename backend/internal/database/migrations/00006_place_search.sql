-- +goose Up
CREATE TABLE place_search_cache (
    cache_key text PRIMARY KEY CHECK (length(cache_key) = 64),
    results jsonb NOT NULL,
    result_count integer NOT NULL CHECK (result_count >= 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX place_search_cache_expiry_idx ON place_search_cache (expires_at);

CREATE TABLE provider_daily_usage (
    provider text NOT NULL,
    usage_date date NOT NULL,
    request_count integer NOT NULL CHECK (request_count >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, usage_date)
);

-- +goose Down
DROP TABLE IF EXISTS provider_daily_usage;
DROP TABLE IF EXISTS place_search_cache;

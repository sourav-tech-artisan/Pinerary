-- +goose Up
CREATE TABLE background_jobs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_type text NOT NULL,
    payload jsonb NOT NULL,
    idempotency_key text,
    state text NOT NULL DEFAULT 'available'
        CHECK (state IN ('available', 'running', 'dead')),
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 8 CHECK (max_attempts > 0),
    run_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    locked_by text,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX background_jobs_idempotency_idx
    ON background_jobs (job_type, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX background_jobs_poll_idx
    ON background_jobs (run_at, id) WHERE state = 'available';

CREATE TABLE outbox_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS background_jobs;

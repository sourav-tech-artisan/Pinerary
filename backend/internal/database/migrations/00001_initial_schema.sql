-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oidc_subject text NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT '',
    default_transport_mode text NOT NULL DEFAULT 'motorcycle'
        CHECK (default_transport_mode IN ('motorcycle', 'car', 'walking')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    installation_id text NOT NULL,
    platform text NOT NULL CHECK (platform IN ('web', 'android')),
    push_subscription jsonb,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, installation_id)
);

CREATE TABLE journeys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_request_id uuid NOT NULL,
    kind text NOT NULL CHECK (kind IN ('trip', 'outing')),
    label text NOT NULL CHECK (char_length(label) BETWEEN 1 AND 160),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended', 'expired')),
    started_at timestamptz NOT NULL DEFAULT now(),
    ended_at timestamptz,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, client_request_id),
    CHECK ((status = 'active' AND ended_at IS NULL) OR (status <> 'active' AND ended_at IS NOT NULL))
);

CREATE UNIQUE INDEX journeys_one_active_per_owner_idx
    ON journeys (owner_id) WHERE status = 'active';
CREATE INDEX journeys_owner_started_idx ON journeys (owner_id, started_at DESC);

CREATE TABLE places (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    notes text NOT NULL DEFAULT '',
    location geography(Point, 4326) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX places_owner_idx ON places (owner_id) WHERE deleted_at IS NULL;
CREATE INDEX places_location_gist_idx ON places USING gist (location);

CREATE TABLE journey_stops (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    journey_id uuid NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    place_id uuid REFERENCES places(id) ON DELETE SET NULL,
    sequence_number bigint NOT NULL,
    captured_at timestamptz NOT NULL,
    display_name text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 200),
    note text NOT NULL DEFAULT '',
    location geography(Point, 4326) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (journey_id, sequence_number)
);

CREATE INDEX journey_stops_journey_timeline_idx
    ON journey_stops (journey_id, sequence_number);

CREATE TABLE location_samples (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    journey_id uuid NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sample_id uuid NOT NULL,
    captured_at timestamptz NOT NULL,
    location geography(Point, 4326) NOT NULL,
    accuracy_m real,
    speed_mps real,
    heading_deg real,
    is_accepted boolean NOT NULL DEFAULT true,
    rejection_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (journey_id, sample_id)
);

CREATE INDEX location_samples_journey_time_idx
    ON location_samples (journey_id, captured_at);
CREATE INDEX location_samples_location_gist_idx
    ON location_samples USING gist (location);

CREATE TABLE route_segments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    journey_id uuid NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    segment_number integer NOT NULL,
    started_at timestamptz NOT NULL,
    ended_at timestamptz NOT NULL,
    raw_path geography(LineString, 4326),
    matched_path geography(LineString, 4326),
    distance_m double precision NOT NULL DEFAULT 0,
    duration_s integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (journey_id, segment_number)
);

CREATE TABLE photos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    journey_id uuid REFERENCES journeys(id) ON DELETE CASCADE,
    stop_id uuid REFERENCES journey_stops(id) ON DELETE SET NULL,
    object_key text NOT NULL UNIQUE,
    thumbnail_key text,
    content_type text NOT NULL,
    byte_size bigint,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'uploaded', 'processed', 'failed')),
    captured_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (journey_id IS NOT NULL OR stop_id IS NULL)
);

CREATE INDEX photos_journey_idx ON photos (journey_id, created_at);
CREATE INDEX photos_stop_idx ON photos (stop_id, created_at);

CREATE TABLE itinerary_shares (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    journey_id uuid NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    snapshot jsonb NOT NULL,
    expires_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX itinerary_shares_journey_idx ON itinerary_shares (journey_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS itinerary_shares;
DROP TABLE IF EXISTS photos;
DROP TABLE IF EXISTS route_segments;
DROP TABLE IF EXISTS location_samples;
DROP TABLE IF EXISTS journey_stops;
DROP TABLE IF EXISTS places;
DROP TABLE IF EXISTS journeys;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS postgis;
DROP EXTENSION IF EXISTS pgcrypto;

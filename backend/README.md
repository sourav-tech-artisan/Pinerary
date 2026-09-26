# Pinerary Backend

The backend is a Go modular monolith with three executables:

- `pinerary-api` serves the authenticated REST API and public itinerary pages.
- `pinerary-worker` cleans and map-matches routes, processes photos, and enforces outing deadlines.
- `pinerary-migrate` applies or inspects embedded Goose migrations.

PostgreSQL/PostGIS is the source of truth, MinIO provides local S3-compatible storage, Geoapify provides advance-place suggestions, Nominatim provides reverse geocoding, and Valhalla provides road matrices and map matching.

## Requirements

- Go 1.24+
- Docker with a running Compose-compatible daemon
- Docker storage for the local Valhalla graph

The Compose file starts PostgreSQL/PostGIS, MinIO, and Valhalla. The initial Valhalla graph uses BBBike's compact New Delhi extract and is persisted in the `valhalla-data` Docker volume. The first start downloads the image/data and builds the graph, so it takes longer than subsequent starts.

The official PostGIS image is pinned to `linux/amd64`; Docker Desktop uses emulation on Apple Silicon. The first pull/start can therefore take longer, but subsequent starts reuse the image and volumes.

## Local setup

From `backend/`:

```bash
cp .env.example .env
set -a
. ./.env
set +a
make db-up
make migrate
```

The example credentials are local-development defaults only. Before using public Nominatim, replace the placeholder contact in `PINERARY_NOMINATIM_USER_AGENT` and follow its current usage policy.

Place suggestions are disabled until a Geoapify key is configured. Create a free server-side key, then set:

```bash
PINERARY_PLACE_SEARCH_PROVIDER=geoapify
PINERARY_GEOAPIFY_API_KEY=<your-key>
```

Keep this key in `backend/.env` or a production secret store, never in `NEXT_PUBLIC_*` configuration or Git.

Start the API and worker in separate terminals after loading `.env` in each:

```bash
make run
```

```bash
make run-worker
```

The API listens at `http://localhost:8080`. Useful endpoints are:

- `GET /health/live`
- `GET /health/ready`
- `GET /metrics`
- `GET /openapi.yaml`

The example environment now uses the configured Auth0 tenant. Log in through the PWA to obtain an API access token.

For isolated Postman/backend development, change `PINERARY_AUTH_MODE` to `development`; any non-empty bearer token then becomes a stable local identity. For example:

```bash
curl -H 'Authorization: Bearer alice' http://localhost:8080/api/v1/me
```

Never enable `PINERARY_AUTH_MODE=development` in a public environment. Production uses `oidc` and requires `PINERARY_OIDC_ISSUER_URL` plus `PINERARY_OIDC_AUDIENCE`.

## External services

- MinIO console: `http://localhost:9001`
- MinIO S3 endpoint: `http://localhost:9000`
- Valhalla default: `http://localhost:8002`
- Geoapify default: `https://api.geoapify.com` when enabled
- Nominatim default: `https://nominatim.openstreetmap.org`

Self-hosted Valhalla does not require an account or API key. It builds its graph from the configured OpenStreetMap PBF extract. To add another disconnected region such as Goa, add its PBF URL to `tile_urls` and recreate Valhalla; file hashes trigger a graph rebuild while the backend endpoint remains unchanged. Public Nominatim also has no API key, but its usage policy requires an identifying application/contact and permits only light user-triggered traffic.

Geoapify autocomplete requests are proxied through the API, cached by a hash of the normalized query, limited to 30 requests per user per minute, and capped at 2,500 upstream calls per UTC day by default. The application cap deliberately stays below the provider's current free allowance. The API returns `429 place_search_quota_exhausted` after reaching it.

If Valhalla is unavailable, normal capture still works; nearby road ranking returns `503`, and queued map-matching jobs retry before entering the dead-letter state. Straight-line distance is only used to shortlist candidates, never as the successful default nearby result. The worker also removes photo uploads that remain pending for more than 24 hours.

Authenticated place-suggestion, reverse-geocode, and upload requests are limited to 30 per minute per user; nearby requests allow 60. Public share reads are limited to 120 per minute per client IP. These in-process limits are per API replica; use an edge limiter as an additional production control when horizontally scaling.

Generate Web Push credentials with:

```bash
make vapid
```

Set the printed public/private values and a `mailto:` or HTTPS contact in the corresponding VAPID environment variables. Empty credentials disable successful push delivery but do not prevent the 24-hour server-side outing expiry.

## Build and test

```bash
make build
make fmt-check
make test
make vet
make lint
```

Run `make generate` after changing migrations or SQL queries. Generated `sqlc` code is committed; CI regenerates it and rejects drift.

The PostGIS tests require a migrated database:

```bash
PINERARY_DATABASE_URL="$PINERARY_DATABASE_URL" go test -tags=integration ./internal/integration
```

## Container

Build one image containing all three runtime binaries:

```bash
docker build -t pinerary-backend .
```

The default entrypoint is `/pinerary-api`. Override it for the other roles:

```bash
docker run --rm --env-file .env --entrypoint /pinerary-worker pinerary-backend
docker run --rm --env-file .env --entrypoint /pinerary-migrate pinerary-backend up
```

Run migrations as a deployment job before replacing API/worker instances. Do not run `down` automatically in production.

See the [API handbook](../docs/backend-api.md) for workflows and the [operations handbook](../docs/operations.md) for deployment, recovery, and backups.
The [backend detailed design](../docs/backend-detailed-design.md) explains the repository package-by-package with sequence, state, data, and processing diagrams. A ready-to-import Postman collection and local environment are under [`../docs/postman`](../docs/postman).

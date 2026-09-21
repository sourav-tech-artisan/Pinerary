# Pinerary Backend

The backend is a Go modular monolith with three executables:

- `pinerary-api` serves the authenticated REST API and public itinerary pages.
- `pinerary-worker` cleans and map-matches routes, processes photos, and enforces outing deadlines.
- `pinerary-migrate` applies or inspects embedded Goose migrations.

PostgreSQL/PostGIS is the source of truth, MinIO provides local S3-compatible storage, Nominatim provides reverse geocoding, and Valhalla provides road matrices and map matching.

## Requirements

- Go 1.24+
- Docker with a running Compose-compatible daemon
- A Valhalla endpoint for nearby road results and map matching

The Compose file starts PostgreSQL/PostGIS and MinIO. Valhalla is deliberately external because its image and regional routing graph depend on the deployment region.

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

Development auth treats any non-empty bearer token as a stable local identity. For example:

```bash
curl -H 'Authorization: Bearer alice' http://localhost:8080/api/v1/me
```

Never enable `PINERARY_AUTH_MODE=development` in a public environment. Production uses `oidc` and requires `PINERARY_OIDC_ISSUER_URL` plus `PINERARY_OIDC_AUDIENCE`.

## External services

- MinIO console: `http://localhost:9001`
- MinIO S3 endpoint: `http://localhost:9000`
- Valhalla default: `http://localhost:8002`
- Nominatim default: `https://nominatim.openstreetmap.org`

Self-hosted Valhalla does not require an account or API key. It requires an OpenStreetMap PBF extract and a locally built graph for the chosen region. Public Nominatim also has no API key, but its usage policy requires an identifying application/contact and permits only light user-triggered traffic.

If Valhalla is unavailable, normal capture still works; nearby road ranking returns `503`, and queued map-matching jobs retry before entering the dead-letter state. Straight-line distance is only used to shortlist candidates, never as the successful default nearby result. The worker also removes photo uploads that remain pending for more than 24 hours.

Authenticated reverse-geocode/upload requests are limited to 30 per minute per user and nearby requests to 60. Public share reads are limited to 120 per minute per client IP. These in-process limits are per API replica; use an edge limiter as an additional production control when horizontally scaling.

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

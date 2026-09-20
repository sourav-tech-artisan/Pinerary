# Pinerary Backend Implementation and Commit Plan

| Field | Value |
| --- | --- |
| Status | Proposed implementation sequence |
| Last updated | 2026-09-21 |
| Scope | Backend system only |
| Backend root | `backend/` |

## 1. Approach

The backend will be developed before the PWA, in small commits that remain buildable and testable independently. Each commit should introduce one coherent architectural capability or product behaviour.

The existing architecture documents should be committed as a documentation baseline before implementation begins. This is separate from the numbered backend commits below; the first backend implementation commit remains the Gin project setup.

No commit should contain unrelated formatting, speculative abstractions, disabled tests, secrets, or unfinished code hidden behind comments.

## 2. Initial repository layout

```text
Pinerary/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   └── worker/
│   ├── internal/
│   │   ├── identity/
│   │   ├── journeys/
│   │   ├── places/
│   │   ├── tracking/
│   │   ├── nearby/
│   │   ├── media/
│   │   ├── sharing/
│   │   ├── notifications/
│   │   ├── jobs/
│   │   └── platform/
│   ├── migrations/
│   ├── queries/
│   ├── api/
│   ├── go.mod
│   └── go.sum
├── docs/
└── infra/
```

Directories should be created only when their first real code is introduced. Empty architecture-shaped packages are not useful.

## 3. Commit rules

Every commit must:

1. Build successfully.
2. Pass unit/integration tests relevant to its scope.
3. Pass `go test`, `go vet`, formatting, and configured lint checks.
4. Include migrations and API-contract changes required by the feature.
5. Preserve backward compatibility unless the commit explicitly documents a pre-release contract change.
6. Avoid mixing mechanical refactors with behavioural changes.
7. Use a Conventional Commit-style subject.

Before committing, review the exact staged diff and exclude generated binaries, local databases, credentials, uploaded files, and environment-specific configuration.

## 4. Backend phases and commits

### Phase A — Service foundation

#### Commit 1: `chore(backend): bootstrap Go API with Gin`

Scope:

- Initialize the Go module under `backend/`.
- Add Gin and a minimal router.
- Add `cmd/api/main.go`.
- Add environment-based HTTP address configuration.
- Add `GET /health/live`.
- Add graceful shutdown for `SIGINT` and `SIGTERM`.
- Add a router/health-handler test.
- Add minimal backend run instructions.

Explicitly excluded:

- PostgreSQL
- Authentication
- OpenAPI
- Docker Compose
- Domain packages
- Background workers

This keeps the first commit small and proves the process lifecycle and test structure.

#### Commit 2: `chore(backend): add quality checks and continuous integration`

Scope:

- Add formatting, test, vet, and lint commands.
- Add a backend `Makefile` or equivalent task entry points.
- Add CI for the Go backend.
- Add a multi-stage API Dockerfile.
- Add `.gitignore` entries for Go build/test artifacts and local configuration.

#### Commit 3: `feat(api): add production HTTP middleware and error model`

Scope:

- Structured `slog` logging.
- Request/correlation IDs.
- Panic recovery.
- Request-size and server timeout limits.
- Configurable CORS for the future PWA origin.
- RFC 9457-style problem responses.
- Redaction rules preventing tokens and coordinates from entering routine logs.

#### Commit 4: `docs(api): define the initial OpenAPI contract`

Scope:

- Add the versioned `/v1` API contract.
- Define common IDs, timestamps, pagination, problem responses, and idempotency headers.
- Add OpenAPI validation/linting to CI.
- Add only health and shared schemas initially; domain endpoints arrive with their features.

### Phase B — Persistence and platform services

#### Commit 5: `feat(storage): add PostgreSQL and PostGIS foundation`

Scope:

- Add local PostgreSQL/PostGIS development infrastructure.
- Add SQL-first Goose migrations.
- Add `pgx` connection-pool configuration and shutdown.
- Add `GET /health/ready` with database readiness.
- Add integration-test infrastructure using real PostGIS.

#### Commit 6: `feat(storage): add sqlc queries and transaction boundary`

Scope:

- Configure `sqlc` with `pgx` types.
- Add transaction helpers that preserve `context.Context`.
- Establish conventions for repositories and module-owned queries.
- Add migration and generated-query checks to CI.

#### Commit 7: `feat(jobs): add durable jobs and transactional outbox`

Scope:

- Add `jobs` and `outbox_events` tables.
- Add leasing with `FOR UPDATE SKIP LOCKED`.
- Add retry/backoff and dead-letter states.
- Add `cmd/worker/main.go` using the same platform configuration.
- Add concurrency and recovery integration tests.

The queue is introduced before features need asynchronous work, avoiding ad-hoc goroutines in request handlers.

### Phase C — Identity and authorization

#### Commit 8: `feat(identity): add OIDC authentication boundary`

Scope:

- Validate issuer, audience, signature, and expiry for OIDC access tokens.
- Map external issuer/subject pairs to internal user UUIDs.
- Add authentication middleware and request principal.
- Add a test-only issuer/JWKS fixture.
- Add ownership-check conventions for repositories/services.

Decision gate: select the production OIDC provider before deploying this commit, but keep the backend implementation provider-neutral.

#### Commit 9: `feat(identity): add devices and user preferences`

Scope:

- Add device registrations.
- Add user preferences, including default nearby mode `motorcycle`.
- Add device/application version and last-seen metadata.
- Add authenticated profile/preferences endpoints.

### Phase D — Journeys and places

#### Commit 10: `feat(journeys): add journey domain and persistence`

Scope:

- Add `journeys` schema and repository.
- Implement `draft -> active -> completed` transitions.
- Enforce one active journey per user.
- Support outing-to-trip conversion.
- Enforce completed-timeline immutability in domain tests.

#### Commit 11: `feat(journeys): expose journey lifecycle API`

Scope:

- Create, read, list, activate, convert, and complete journeys.
- Add optimistic versions.
- Add idempotency for retryable mutations.
- Extend OpenAPI and add HTTP/integration tests.

#### Commit 12: `feat(places): add saved places and journey stops`

Scope:

- Add `places`, `user_places`, and `journey_stops` schemas.
- Use PostGIS `geography(Point, 4326)` and GiST indexes.
- Add standalone place saving and journey-stop capture.
- Preserve location snapshots and original timeline order.
- Permit only allowed name/photo edits after completion.

#### Commit 13: `feat(places): add reverse-geocoding adapter`

Scope:

- Define the `Geocoder` port.
- Implement a rate-limited Nominatim adapter.
- Add compliant application identification and result caching.
- Keep user-entered names authoritative over suggestions.
- Add timeout, retry, and provider-failure tests.

### Phase E — Route capture and processing

#### Commit 14: `feat(tracking): add idempotent GPS batch ingestion`

Scope:

- Add `tracks` and append-only `track_points` tables.
- Accept bounded GPS batches.
- Deduplicate with `(track_id, device_id, client_sequence)`.
- Accept late/out-of-order batches safely.
- Store reported accuracy, speed, heading, and client/server times.
- Add ingestion and retry integration tests.

This supports foreground PWA tracking first; it does not introduce Android background collection yet.

#### Commit 15: `feat(tracking): add route cleaning and segmentation worker`

Scope:

- Flag stale, inaccurate, duplicate, and implausible samples.
- Split traces across large gaps or impossible jumps.
- Preserve raw samples and processing version.
- Produce a cleaned PostGIS `LineString`.
- Trigger processing through the transactional outbox.

#### Commit 16: `feat(tracking): add Valhalla map matching`

Scope:

- Define the `MapMatcher` port.
- Add the Valhalla adapter with timeouts and bounded payloads.
- Store map-matched geometry separately from cleaned geometry.
- Fall back to the cleaned track when matching fails.
- Add stubbed-provider integration tests.

### Phase F — Media and nearby discovery

#### Commit 17: `feat(media): add private photo upload workflow`

Scope:

- Add S3-compatible object-store abstraction and MinIO development service.
- Add upload reservation, presigned upload, and completion endpoints.
- Validate ownership, object keys, content types, sizes, magic bytes, and checksums.
- Add abandoned-upload cleanup jobs.
- Keep buckets private.

#### Commit 18: `feat(media): add thumbnail processing`

Scope:

- Generate normalized thumbnails asynchronously.
- Record dimensions and processing state.
- Strip unnecessary EXIF data from derived images.
- Add retry/failure handling and tests.

#### Commit 19: `feat(nearby): add road-ranked nearby place search`

Scope:

- Use PostGIS only to create an internal candidate shortlist.
- Define the `RouteMatrix` port.
- Query Valhalla for car, motorcycle, and walking matrices.
- Sort by motorcycle road distance by default.
- Remember the user's latest selected mode.
- Return a clearly labelled straight-line fallback when routing is unavailable.
- Add spatial query-plan and partial-provider-failure tests.

### Phase G — Sharing and outing lifecycle

#### Commit 20: `feat(sharing): add itinerary snapshots and public links`

Scope:

- Add share exports, ordered export items, and share links.
- Store only hashes of high-entropy public tokens.
- Add explicit privacy selections for notes, photos, route, and coordinates.
- Add revocation and optional expiration.
- Render responsive public HTML from Go with Open Graph metadata.
- Prevent indexing and referrer leakage.

#### Commit 21: `feat(notifications): add outing expiry workflow`

Scope:

- Schedule the 23-hour warning and 24-hour completion transactionally.
- Add Web Push subscription storage and sender abstraction.
- Complete outings even if notification delivery fails.
- Add clock-controlled lifecycle and retry tests.

### Phase H — Backend hardening

#### Commit 22: `feat(observability): add metrics and distributed tracing`

Scope:

- Add OpenTelemetry-compatible traces and metrics.
- Instrument HTTP, PostgreSQL, jobs, uploads, and provider adapters.
- Ensure sensitive coordinates, tokens, and signed URLs are redacted.
- Add operational dashboards/queries as documentation.

#### Commit 23: `test(backend): add end-to-end and performance baselines`

Scope:

- Add end-to-end tests for the primary backend workflows.
- Add authorization-isolation tests for independent users.
- Add baseline load tests for GPS ingestion and nearby queries.
- Record PostGIS query plans and latency budgets.
- Add account data deletion/export coverage.

#### Commit 24: `docs(backend): finalize operations and API handbook`

Scope:

- Document configuration and secret management.
- Document local startup, migration, backup, restore, and worker recovery.
- Document external-service attribution and usage constraints.
- Reconcile OpenAPI and architecture documents with the implemented system.

## 5. After the backend

Once the backend contract is stable enough for client work:

1. Build the installable PWA against the OpenAPI contract.
2. Add offline IndexedDB synchronization and foreground tracking.
3. Complete and harden all PWA flows.
4. In the final project phase, package the static Next.js output with Capacitor Android and add background tracking, Android native storage adapters, FCM, and signed APK distribution.

The Android Capacitor work must not be pulled into an earlier backend commit.

## 6. First execution checkpoint

The first implementation checkpoint is Commit 1 only:

```text
chore(backend): bootstrap Go API with Gin
```

Stop after its tests pass and review the resulting structure before proceeding to CI, Docker, databases, or domain code.

# Backend Operations Handbook

## Runtime topology

Deploy one container image in three roles:

```text
migration job -> PostgreSQL/PostGIS
API           -> PostgreSQL, object storage, Nominatim, Valhalla
worker        -> PostgreSQL, object storage, Valhalla, Web Push
```

The API and worker may be replicated independently. PostgreSQL job leasing uses `FOR UPDATE SKIP LOCKED`, so multiple workers can safely compete for work. Keep at least one worker running; the API does not perform asynchronous work in request goroutines. An hourly job removes photo reservations that remain pending for more than 24 hours.

## Required configuration

| Variable | Production requirement |
| --- | --- |
| `PINERARY_DATABASE_URL` | PostgreSQL connection with PostGIS available; use TLS where supported |
| `PINERARY_AUTH_MODE` | `oidc` |
| `PINERARY_OIDC_ISSUER_URL` | HTTPS issuer supporting OIDC discovery |
| `PINERARY_OIDC_AUDIENCE` | API audience/client ID expected in tokens |
| `PINERARY_ALLOWED_ORIGINS` | Comma-separated exact PWA origins |
| `PINERARY_PUBLIC_BASE_URL` | Public HTTPS origin used in share links |
| `PINERARY_OBJECT_*` | Private S3-compatible endpoint, credentials, bucket, and TLS flag |
| `PINERARY_VALHALLA_URL` | Private/restricted regional Valhalla endpoint |
| `PINERARY_NOMINATIM_*` | Provider URL and compliant identifying user agent |
| `PINERARY_VAPID_*` | Web Push contact, public key, and private key |
| `PINERARY_OTEL_EXPORTER_OTLP_ENDPOINT` | Optional OTLP HTTP traces endpoint |

Keep secrets in the hosting platform's secret manager, not an image, repository, logs, or frontend environment. Rotate database, object-store, and VAPID credentials deliberately; existing browser push subscriptions may need renewal after a VAPID rotation.

## Deployment order

1. Back up PostgreSQL before a risky schema release.
2. Run `/pinerary-migrate status` against the target database.
3. Run `/pinerary-migrate up` as a single deployment job.
4. Replace API instances and verify `/health/live` and `/health/ready`.
5. Replace worker instances and check that the job backlog drains.
6. Smoke-test OIDC, place save, nearby routing, photo upload, and a revocable share.

Migrations are embedded in the binary. Never automate `migrate down` in production; forward-fix unless a reviewed rollback explicitly proves data safety.

## Health and telemetry

- `/health/live` proves the API process can serve HTTP.
- `/health/ready` proves PostgreSQL is reachable and should gate traffic.
- `/metrics` exposes Prometheus-format HTTP counters and latency.
- JSON logs include request/trace IDs but omit request bodies, coordinates, tokens, and signed URLs.
- OTLP trace export is optional; leaving the endpoint empty keeps local no-op export.

Recommended alerts:

- Readiness failures or elevated 5xx rate
- API p95 latency regression
- `background_jobs` in `dead` state
- Old available/running jobs and a growing queue
- Valhalla/Nominatim error or latency spikes
- Object-storage failures
- Database connection saturation and storage growth

Costly authenticated endpoints have per-user in-process limits, and public share reads have a per-IP limit. These limits apply independently to each API replica. Add gateway/CDN limits when aggregate enforcement or stronger denial-of-service protection is required, and configure trusted proxy handling explicitly there.

## Job recovery

Workers lease jobs. On startup, work left `running` for more than 15 minutes is returned to the available queue. Handler failures retry with exponential backoff until `max_attempts`, then become `dead`.

Inspect without exposing payloads unnecessarily:

```sql
SELECT job_type, state, count(*)
FROM background_jobs
GROUP BY job_type, state
ORDER BY job_type, state;

SELECT id, job_type, attempts, max_attempts, run_at, left(last_error, 300) AS error
FROM background_jobs
WHERE state = 'dead'
ORDER BY updated_at DESC;
```

Before replaying a dead job, fix the cause and confirm the handler is idempotent. Then reset only explicitly reviewed IDs to `available`, clear their lease, and choose whether to reset attempts. Do not mass-requeue unknown failures.

## Backup and restore

PostgreSQL is the metadata source of truth; object storage contains originals and thumbnails. Protect both on a coordinated schedule.

Example logical database backup:

```bash
pg_dump --format=custom --no-owner --file=pinerary.dump "$PINERARY_DATABASE_URL"
```

Example restore into an empty recovery database:

```bash
pg_restore --no-owner --clean --if-exists --dbname="$RECOVERY_DATABASE_URL" pinerary.dump
```

Use the object provider's versioning/snapshot mechanism for the private media bucket. A database-only restore can reference missing objects; an object-only restore lacks ownership and lifecycle metadata. Regularly test restoration in an isolated environment, run migration status, compare representative photo keys, and verify PostGIS queries.

## Provider and failure behaviour

- **Valhalla:** host a regional OpenStreetMap graph. Nearby road ranking returns `503` when the selected mode fails. Map-matching jobs retry, while cleaned routes remain available.
- **Nominatim:** requests are user-triggered, cached, and identified. Verify the current public usage policy before deployment; self-host or replace it when usage no longer fits.
- **Object storage:** buckets stay private. Clients receive short-lived PUT/GET URLs. Configure CORS only for the PWA origins and allowed methods.
- **Web Push:** notification delivery is best effort. Outing expiration is a database job and proceeds even if push delivery fails.
- **Map tiles:** the frontend must provide attribution and obey the selected tile provider's current policy. Offline user data does not imply bulk offline tile downloads.

## Privacy and retention

GPS routes and photos are sensitive. Use HTTPS, encrypted managed volumes/object storage, least-privilege service credentials, restricted database access, and backups with the same retention controls as production. Public itinerary links are unlisted but are bearer secrets: anyone with a link can view its snapshot until expiry or revocation.

Raw GPS/photo retention and account export/deletion policy remain product decisions before public launch. Until implemented, do not describe the service as supporting self-service deletion or a fixed retention guarantee.

## Local incident checks

1. Confirm API readiness and database connectivity.
2. Confirm worker logs show polling rather than repeated job errors.
3. Check the job state counts above.
4. Check MinIO/S3 reachability and bucket permissions.
5. Probe Valhalla independently for the deployed regional graph.
6. Verify OIDC discovery/JWKS reachability and system clock accuracy.
7. Use request/trace IDs to correlate a failure without logging sensitive payloads.

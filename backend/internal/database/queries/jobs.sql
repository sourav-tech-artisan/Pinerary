-- name: EnqueueJob :one
INSERT INTO background_jobs (job_type, payload, idempotency_key, max_attempts, run_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (job_type, idempotency_key) WHERE idempotency_key IS NOT NULL
DO UPDATE SET updated_at = background_jobs.updated_at
RETURNING *;

-- name: ClaimNextJobs :many
WITH next_jobs AS (
    SELECT id
    FROM background_jobs
    WHERE state = 'available' AND run_at <= now()
    ORDER BY run_at, id
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
)
UPDATE background_jobs AS jobs
SET state = 'running',
    attempts = attempts + 1,
    locked_at = now(),
    locked_by = sqlc.arg(worker_id),
    updated_at = now()
FROM next_jobs
WHERE jobs.id = next_jobs.id
RETURNING jobs.*;

-- name: CompleteJob :exec
DELETE FROM background_jobs
WHERE id = $1 AND state = 'running';

-- name: RetryJob :exec
UPDATE background_jobs
SET state = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'available' END,
    run_at = CASE WHEN attempts >= max_attempts THEN run_at ELSE $2 END,
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    updated_at = now()
WHERE id = $1 AND state = 'running';

-- name: RequeueStaleJobs :execrows
UPDATE background_jobs
SET state = 'available',
    locked_at = NULL,
    locked_by = NULL,
    run_at = now(),
    updated_at = now()
WHERE state = 'running' AND locked_at < $1;

-- name: AddOutboxEvent :one
INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListUnpublishedEvents :many
SELECT * FROM outbox_events
WHERE published_at IS NULL
ORDER BY id
LIMIT $1;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events SET published_at = now() WHERE id = $1;

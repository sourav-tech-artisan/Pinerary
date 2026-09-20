-- name: UpsertDevice :one
INSERT INTO devices (user_id, installation_id, platform, push_subscription)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, installation_id) DO UPDATE
SET platform = EXCLUDED.platform,
    push_subscription = EXCLUDED.push_subscription,
    last_seen_at = now(),
    updated_at = now()
RETURNING *;

-- name: ListDevices :many
SELECT * FROM devices
WHERE user_id = $1
ORDER BY last_seen_at DESC;

-- name: DeleteDevice :execrows
DELETE FROM devices WHERE id = $1 AND user_id = $2;

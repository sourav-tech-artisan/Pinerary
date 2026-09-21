-- name: GetJourneyByID :one
SELECT * FROM journeys WHERE id = $1;

-- name: ListPushSubscriptionsForUser :many
SELECT id, push_subscription
FROM devices
WHERE user_id = $1 AND push_subscription IS NOT NULL;

-- name: ClearDevicePushSubscription :exec
UPDATE devices SET push_subscription = NULL, updated_at = now() WHERE id = $1;

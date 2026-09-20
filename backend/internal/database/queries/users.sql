-- name: GetUserByID :one
SELECT id, oidc_subject, display_name, default_transport_mode, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserBySubject :one
SELECT id, oidc_subject, display_name, default_transport_mode, created_at, updated_at
FROM users
WHERE oidc_subject = $1;

-- name: UpsertUserFromIdentity :one
INSERT INTO users (oidc_subject, display_name)
VALUES ($1, $2)
ON CONFLICT (oidc_subject) DO UPDATE
SET display_name = CASE
        WHEN EXCLUDED.display_name = '' THEN users.display_name
        ELSE EXCLUDED.display_name
    END,
    updated_at = now()
RETURNING id, oidc_subject, display_name, default_transport_mode, created_at, updated_at;

-- name: UpdateUserPreferences :one
UPDATE users
SET display_name = $2,
    default_transport_mode = $3,
    updated_at = now()
WHERE id = $1
RETURNING id, oidc_subject, display_name, default_transport_mode, created_at, updated_at;

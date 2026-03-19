-- name: UpsertUser :one
INSERT INTO users (id, username, first_name, last_name) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: IsUserBlocked :one
SELECT is_blocked FROM users WHERE id = $1;

-- name: ListAllActiveUsers :many
SELECT * FROM users WHERE is_blocked = FALSE;

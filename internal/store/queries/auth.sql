-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: UpdateUser :one
UPDATE users SET name = ?, email = ?, password_hash = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: CreateAPIKey :one
INSERT INTO api_keys (id, user_id, name, key_hash, permissions, created_at, last_used_at)
VALUES (?, ?, ?, ?, ?, ?, NULL)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = ?;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = ?;

-- name: UpdateAPIKeyName :one
UPDATE api_keys SET name = ? WHERE id = ?
RETURNING *;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = ? WHERE id = ?;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = ?;

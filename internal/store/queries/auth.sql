-- name: CreateAPIKey :one
INSERT INTO api_keys (id, user_id, name, key_hash, permissions, scope, expires_at, created_at, last_used_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = ? AND (expires_at IS NULL OR expires_at > datetime('now'));

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = ?;

-- name: UpdateAPIKeyName :one
UPDATE api_keys SET name = ? WHERE id = ?
RETURNING *;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = ? WHERE id = ?;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = ?;

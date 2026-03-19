-- name: CreateArea :one
INSERT INTO areas (
    id, title, "index", archived, deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, 0, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetArea :one
SELECT * FROM areas
WHERE id = ? AND deleted = 0;

-- name: ListAreas :many
SELECT * FROM areas
WHERE deleted = 0 AND archived = 0
ORDER BY "index";

-- name: ListAllAreas :many
SELECT * FROM areas
WHERE deleted = 0
ORDER BY "index";

-- name: UpdateAreaTitle :one
UPDATE areas SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateAreaArchived :one
UPDATE areas SET archived = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteArea :exec
UPDATE areas SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

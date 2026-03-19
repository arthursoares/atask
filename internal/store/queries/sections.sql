-- name: CreateSection :one
INSERT INTO sections (
    id, title, project_id, "index", deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetSection :one
SELECT * FROM sections
WHERE id = ? AND deleted = 0;

-- name: ListSectionsByProject :many
SELECT * FROM sections
WHERE project_id = ? AND deleted = 0
ORDER BY "index";

-- name: UpdateSectionTitle :one
UPDATE sections SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteSection :exec
UPDATE sections SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

-- name: SoftDeleteSectionsByProject :exec
UPDATE sections SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE project_id = ? AND deleted = 0;

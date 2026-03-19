-- name: CreateTag :one
INSERT INTO tags (
    id, title, parent_id, shortcut, "index", deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tags
WHERE id = ? AND deleted = 0;

-- name: ListTags :many
SELECT * FROM tags
WHERE deleted = 0
ORDER BY "index";

-- name: UpdateTagTitle :one
UPDATE tags SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTagShortcut :one
UPDATE tags SET shortcut = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteTag :exec
UPDATE tags SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

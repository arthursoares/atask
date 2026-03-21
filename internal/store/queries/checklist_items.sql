-- name: CreateChecklistItem :one
INSERT INTO checklist_items (
    id, title, status, task_id, "index", deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetChecklistItem :one
SELECT * FROM checklist_items
WHERE id = ? AND deleted = 0;

-- name: ListChecklistItemsByTask :many
SELECT * FROM checklist_items
WHERE task_id = ? AND deleted = 0
ORDER BY "index";

-- name: UpdateChecklistItemTitle :one
UPDATE checklist_items SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateChecklistItemStatus :one
UPDATE checklist_items SET status = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteChecklistItem :exec
UPDATE checklist_items SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

-- name: CountChecklistByTask :one
SELECT
    COUNT(*) AS total,
    COUNT(CASE WHEN status = 1 THEN 1 END) AS done
FROM checklist_items
WHERE task_id = ? AND deleted = 0;

-- name: AddTaskLink :exec
INSERT OR IGNORE INTO task_links (task_id, related_task_id, relationship_type, created_at)
VALUES (?, ?, ?, ?);

-- name: RemoveTaskLink :exec
DELETE FROM task_links WHERE task_id = ? AND related_task_id = ?;

-- name: ListTaskLinks :many
SELECT * FROM task_links WHERE task_id = ?;

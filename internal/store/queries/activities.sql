-- name: CreateActivity :one
INSERT INTO activities (
    id, task_id, actor_id, actor_type, type, content, created_at, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: ListActivitiesByTask :many
SELECT * FROM activities
WHERE task_id = ? AND user_id = ?
ORDER BY created_at DESC;

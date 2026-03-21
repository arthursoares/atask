-- name: CreateTask :one
INSERT INTO tasks (
    id, title, notes, status, schedule, start_date, deadline, completed_at,
    "index", today_index, project_id, section_id, area_id, location_id,
    recurrence_rule, deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? AND deleted = 0;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE deleted = 0
ORDER BY "index";

-- name: ListTasksByProject :many
SELECT * FROM tasks
WHERE project_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksByArea :many
SELECT * FROM tasks
WHERE area_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksBySection :many
SELECT * FROM tasks
WHERE section_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksByLocation :many
SELECT * FROM tasks
WHERE location_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksBySchedule :many
SELECT * FROM tasks
WHERE schedule = ? AND deleted = 0
ORDER BY "index";

-- name: UpdateTaskTitle :one
UPDATE tasks SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskNotes :one
UPDATE tasks SET notes = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskStatus :one
UPDATE tasks SET status = ?, completed_at = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskSchedule :one
UPDATE tasks SET schedule = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskStartDate :one
UPDATE tasks SET start_date = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskDeadline :one
UPDATE tasks SET deadline = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskProject :one
UPDATE tasks SET project_id = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskSection :one
UPDATE tasks SET section_id = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskArea :one
UPDATE tasks SET area_id = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskLocation :one
UPDATE tasks SET location_id = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskRecurrence :one
UPDATE tasks SET recurrence_rule = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskIndex :one
UPDATE tasks SET "index" = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskTodayIndex :one
UPDATE tasks SET today_index = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskTimeSlot :one
UPDATE tasks SET time_slot = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteTask :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

-- name: SoftDeleteTasksByProject :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE project_id = ? AND deleted = 0;

-- name: OrphanTasksByArea :exec
UPDATE tasks SET area_id = NULL, updated_at = ?
WHERE area_id = ? AND deleted = 0;

-- name: OrphanTasksBySection :exec
UPDATE tasks SET section_id = NULL, updated_at = ?
WHERE section_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksByArea :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE area_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksBySection :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE section_id = ? AND deleted = 0;

-- name: CompleteTasksByProject :exec
UPDATE tasks SET status = 1, completed_at = ?, updated_at = ?
WHERE project_id = ? AND status = 0 AND deleted = 0;

-- name: CancelTasksByProject :exec
UPDATE tasks SET status = 2, updated_at = ?
WHERE project_id = ? AND status = 0 AND deleted = 0;

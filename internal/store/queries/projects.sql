-- name: CreateProject :one
INSERT INTO projects (
    id, title, notes, status, schedule, start_date, deadline, completed_at,
    "index", area_id, auto_complete, deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = ? AND deleted = 0;

-- name: ListProjects :many
SELECT * FROM projects
WHERE deleted = 0
ORDER BY "index";

-- name: ListProjectsByArea :many
SELECT * FROM projects
WHERE area_id = ? AND deleted = 0
ORDER BY "index";

-- name: UpdateProjectTitle :one
UPDATE projects SET title = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateProjectNotes :one
UPDATE projects SET notes = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateProjectStatus :one
UPDATE projects SET status = ?, completed_at = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateProjectDeadline :one
UPDATE projects SET deadline = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: UpdateProjectArea :one
UPDATE projects SET area_id = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteProject :exec
UPDATE projects SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

-- name: OrphanProjectsByArea :exec
UPDATE projects SET area_id = NULL, updated_at = ?
WHERE area_id = ? AND deleted = 0;

-- name: CascadeDeleteProjectsByArea :exec
UPDATE projects SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE area_id = ? AND deleted = 0;

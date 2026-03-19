-- name: CreateLocation :one
INSERT INTO locations (
    id, name, latitude, longitude, radius, address, deleted, deleted_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, 0, NULL, ?, ?
)
RETURNING *;

-- name: GetLocation :one
SELECT * FROM locations
WHERE id = ? AND deleted = 0;

-- name: ListLocations :many
SELECT * FROM locations
WHERE deleted = 0;

-- name: UpdateLocationName :one
UPDATE locations SET name = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteLocation :exec
UPDATE locations SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ?;

-- name: ClearLocationFromTasks :exec
UPDATE tasks SET location_id = NULL, updated_at = ?
WHERE location_id = ? AND deleted = 0;

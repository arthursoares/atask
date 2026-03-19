-- name: AddTaskTag :exec
INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?);

-- name: RemoveTaskTag :exec
DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?;

-- name: ListTaskTags :many
SELECT t.* FROM tags t
INNER JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = ? AND t.deleted = 0
ORDER BY t."index";

-- name: RemoveAllTagReferences :exec
DELETE FROM task_tags WHERE tag_id = ?;

-- name: AddProjectTag :exec
INSERT OR IGNORE INTO project_tags (project_id, tag_id) VALUES (?, ?);

-- name: RemoveProjectTag :exec
DELETE FROM project_tags WHERE project_id = ? AND tag_id = ?;

-- name: ListProjectTags :many
SELECT t.* FROM tags t
INNER JOIN project_tags pt ON pt.tag_id = t.id
WHERE pt.project_id = ? AND t.deleted = 0
ORDER BY t."index";

-- name: RemoveAllProjectTagReferences :exec
DELETE FROM project_tags WHERE tag_id = ?;

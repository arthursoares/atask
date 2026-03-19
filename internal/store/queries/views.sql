-- name: ViewInbox :many
SELECT * FROM tasks
WHERE schedule = 0 AND status = 0 AND deleted = 0
ORDER BY "index";

-- name: ViewToday :many
SELECT * FROM tasks
WHERE schedule = 1 AND status = 0 AND deleted = 0
  AND (start_date IS NULL OR start_date <= ?)
ORDER BY COALESCE(today_index, 999999), "index";

-- name: ViewUpcoming :many
SELECT * FROM tasks
WHERE start_date > ? AND status = 0 AND deleted = 0
ORDER BY start_date, "index";

-- name: ViewSomeday :many
SELECT * FROM tasks
WHERE schedule = 2 AND status = 0 AND deleted = 0
ORDER BY "index";

-- name: ViewLogbook :many
SELECT * FROM tasks
WHERE status IN (1, 2) AND deleted = 0
ORDER BY completed_at DESC;

-- name: ViewInbox :many
-- Inbox: unscheduled tasks with no start date set
SELECT * FROM tasks
WHERE schedule = 0 AND status = 0 AND deleted = 0
  AND start_date IS NULL
ORDER BY "index";

-- name: ViewToday :many
-- Today: scheduled as anytime, with no date or start_date <= today
SELECT * FROM tasks
WHERE schedule = 1 AND status = 0 AND deleted = 0
  AND (start_date IS NULL OR start_date <= ?)
ORDER BY
  CASE WHEN time_slot = 'evening' THEN 1 ELSE 0 END,
  COALESCE(today_index, 999999),
  "index";

-- name: ViewUpcoming :many
-- Upcoming: tasks with a future start date (excludes someday)
SELECT * FROM tasks
WHERE start_date > ? AND status = 0 AND deleted = 0
  AND schedule != 2
ORDER BY start_date, "index";

-- name: ViewSomeday :many
-- Someday: long-term, no date, not prioritized
SELECT * FROM tasks
WHERE schedule = 2 AND status = 0 AND deleted = 0
  AND start_date IS NULL
ORDER BY "index";

-- name: ViewLogbook :many
SELECT * FROM tasks
WHERE status IN (1, 2) AND deleted = 0
ORDER BY completed_at DESC;

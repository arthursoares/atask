-- +goose Up

-- Migration 005 added user_id to domain_events but no index covering it.
-- The admin-statistics per-user and time-bucketed queries (internal/store/stats.go)
-- filter/group by (user_id, timestamp) as the event log grows, so index it.
CREATE INDEX idx_domain_events_user_ts ON domain_events(user_id, timestamp);

-- +goose Down
-- Down migration omitted (SQLite makes them painful and rollback is via backup).

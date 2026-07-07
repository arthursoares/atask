-- +goose Up

-- user_id on all domain tables
ALTER TABLE tasks ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE areas ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE activities ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sections ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE checklist_items ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE task_tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE project_tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE task_links ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

-- user_id on event tables
ALTER TABLE delta_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE domain_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

-- Indexes for query performance
CREATE INDEX idx_tasks_user ON tasks(user_id);
CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_areas_user ON areas(user_id);
CREATE INDEX idx_tags_user ON tags(user_id);
CREATE INDEX idx_locations_user ON locations(user_id);
CREATE INDEX idx_sections_user ON sections(user_id);
CREATE INDEX idx_delta_events_user ON delta_events(user_id);

-- Invite tokens for closed-registration flows
CREATE TABLE invites (
    id         TEXT NOT NULL PRIMARY KEY,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'user',
    token      TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    claimed_at DATETIME,
    expires_at DATETIME NOT NULL
);

-- +goose Down
-- Down migrations for SQLite require table recreation; omitted for brevity.
-- In practice, rollback is handled by restoring from backup.

-- +goose Up

CREATE TABLE users (
    id          TEXT NOT NULL PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE api_keys (
    id          TEXT NOT NULL PRIMARY KEY,
    user_id     TEXT REFERENCES users(id),
    name        TEXT,
    key_hash    TEXT UNIQUE,
    permissions TEXT NOT NULL DEFAULT '[]',
    created_at  DATETIME,
    last_used_at DATETIME
);

CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);

CREATE TABLE areas (
    id          TEXT NOT NULL PRIMARY KEY,
    title       TEXT,
    "index"     INTEGER NOT NULL DEFAULT 0,
    archived    INTEGER NOT NULL DEFAULT 0,
    deleted     INTEGER NOT NULL DEFAULT 0,
    deleted_at  DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE tags (
    id          TEXT NOT NULL PRIMARY KEY,
    title       TEXT,
    parent_id   TEXT REFERENCES tags(id),
    shortcut    TEXT,
    "index"     INTEGER NOT NULL DEFAULT 0,
    deleted     INTEGER NOT NULL DEFAULT 0,
    deleted_at  DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE locations (
    id          TEXT NOT NULL PRIMARY KEY,
    name        TEXT,
    latitude    REAL,
    longitude   REAL,
    radius      INTEGER,
    address     TEXT,
    deleted     INTEGER NOT NULL DEFAULT 0,
    deleted_at  DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE projects (
    id           TEXT NOT NULL PRIMARY KEY,
    title        TEXT,
    notes        TEXT NOT NULL DEFAULT '',
    status       INTEGER NOT NULL DEFAULT 0,
    schedule     INTEGER NOT NULL DEFAULT 0,
    start_date   TEXT,
    deadline     TEXT,
    completed_at DATETIME,
    "index"      INTEGER NOT NULL DEFAULT 0,
    area_id      TEXT REFERENCES areas(id),
    auto_complete INTEGER NOT NULL DEFAULT 0,
    deleted      INTEGER NOT NULL DEFAULT 0,
    deleted_at   DATETIME,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE INDEX idx_projects_area_id ON projects (area_id) WHERE deleted = 0;

CREATE TABLE sections (
    id          TEXT NOT NULL PRIMARY KEY,
    title       TEXT,
    project_id  TEXT REFERENCES projects(id),
    "index"     INTEGER NOT NULL DEFAULT 0,
    deleted     INTEGER NOT NULL DEFAULT 0,
    deleted_at  DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE INDEX idx_sections_project_id ON sections (project_id) WHERE deleted = 0;

CREATE TABLE tasks (
    id              TEXT NOT NULL PRIMARY KEY,
    title           TEXT,
    notes           TEXT NOT NULL DEFAULT '',
    status          INTEGER NOT NULL DEFAULT 0,
    schedule        INTEGER NOT NULL DEFAULT 0,
    start_date      TEXT,
    deadline        TEXT,
    completed_at    DATETIME,
    "index"         INTEGER NOT NULL DEFAULT 0,
    today_index     INTEGER,
    project_id      TEXT REFERENCES projects(id),
    section_id      TEXT REFERENCES sections(id),
    area_id         TEXT REFERENCES areas(id),
    location_id     TEXT REFERENCES locations(id),
    recurrence_rule TEXT,
    deleted         INTEGER NOT NULL DEFAULT 0,
    deleted_at      DATETIME,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);

CREATE INDEX idx_tasks_project_id  ON tasks (project_id)  WHERE deleted = 0;
CREATE INDEX idx_tasks_section_id  ON tasks (section_id)  WHERE deleted = 0;
CREATE INDEX idx_tasks_area_id     ON tasks (area_id)     WHERE deleted = 0;
CREATE INDEX idx_tasks_location_id ON tasks (location_id) WHERE deleted = 0;
CREATE INDEX idx_tasks_schedule    ON tasks (schedule)    WHERE deleted = 0;
CREATE INDEX idx_tasks_status      ON tasks (status)      WHERE deleted = 0;
CREATE INDEX idx_tasks_start_date  ON tasks (start_date)  WHERE deleted = 0;
CREATE INDEX idx_tasks_deadline    ON tasks (deadline)    WHERE deleted = 0;

CREATE TABLE checklist_items (
    id          TEXT NOT NULL PRIMARY KEY,
    title       TEXT,
    status      INTEGER NOT NULL DEFAULT 0,
    task_id     TEXT REFERENCES tasks(id),
    "index"     INTEGER NOT NULL DEFAULT 0,
    deleted     INTEGER NOT NULL DEFAULT 0,
    deleted_at  DATETIME,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE INDEX idx_checklist_items_task_id ON checklist_items (task_id) WHERE deleted = 0;

CREATE TABLE task_tags (
    task_id TEXT NOT NULL REFERENCES tasks(id),
    tag_id  TEXT NOT NULL REFERENCES tags(id),
    PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE project_tags (
    project_id TEXT NOT NULL REFERENCES projects(id),
    tag_id     TEXT NOT NULL REFERENCES tags(id),
    PRIMARY KEY (project_id, tag_id)
);

CREATE TABLE task_links (
    task_id           TEXT NOT NULL REFERENCES tasks(id),
    related_task_id   TEXT NOT NULL REFERENCES tasks(id),
    relationship_type TEXT NOT NULL DEFAULT 'related',
    created_at        DATETIME,
    PRIMARY KEY (task_id, related_task_id)
);

CREATE TABLE activities (
    id         TEXT NOT NULL PRIMARY KEY,
    task_id    TEXT REFERENCES tasks(id),
    actor_id   TEXT,
    actor_type TEXT,
    type       TEXT,
    content    TEXT,
    created_at DATETIME
);

CREATE INDEX idx_activities_task_id ON activities (task_id);

CREATE TABLE delta_events (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT,
    entity_id   TEXT,
    action      INTEGER,
    field       TEXT,
    old_value   TEXT,
    new_value   TEXT,
    actor_id    TEXT,
    timestamp   DATETIME
);

CREATE INDEX idx_delta_events_entity ON delta_events (entity_type, entity_id);

CREATE TABLE domain_events (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    type        TEXT,
    entity_type TEXT,
    entity_id   TEXT,
    actor_id    TEXT,
    payload     TEXT NOT NULL DEFAULT '{}',
    timestamp   DATETIME
);

CREATE INDEX idx_domain_events_type   ON domain_events (type);
CREATE INDEX idx_domain_events_entity ON domain_events (entity_type, entity_id);

-- +goose Down

DROP TABLE IF EXISTS domain_events;
DROP TABLE IF EXISTS delta_events;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS task_links;
DROP TABLE IF EXISTS project_tags;
DROP TABLE IF EXISTS task_tags;
DROP TABLE IF EXISTS checklist_items;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS sections;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;

-- +goose Up

-- API keys: retarget user_id to PocketBase user record IDs and add scope/expiry.
-- SQLite cannot ALTER COLUMN to drop the FK reference, so we rebuild the table.
-- Existing rows are preserved; their user_id values become orphaned until
-- `atask admin assign-data --to <pb-user-id>` is run.
CREATE TABLE api_keys_new (
    id           TEXT NOT NULL PRIMARY KEY,
    user_id      TEXT NOT NULL DEFAULT '',
    name         TEXT,
    key_hash     TEXT UNIQUE,
    permissions  TEXT NOT NULL DEFAULT '[]',
    scope        TEXT NOT NULL DEFAULT 'read_write',
    expires_at   DATETIME,
    created_at   DATETIME,
    last_used_at DATETIME
);

INSERT INTO api_keys_new (id, user_id, name, key_hash, permissions, created_at, last_used_at)
SELECT id, COALESCE(user_id, ''), name, key_hash, permissions, created_at, last_used_at
FROM api_keys;

DROP TABLE api_keys;
ALTER TABLE api_keys_new RENAME TO api_keys;
CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);

-- Drop legacy users table; identity now lives in pb_data/data.db
DROP TABLE users;

-- +goose Down
-- Down migrations omitted (SQLite makes them painful and rollback is via backup).

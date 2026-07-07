-- +goose Up

-- Migrations 005/006 added user_id to tags but never rebuilt this index, so
-- tag titles were globally unique across users (a second user could not
-- create a tag with the same title as another user's tag). Rebuild the
-- index scoped to (user_id, title) so uniqueness is per-user again, while
-- still preventing the same user from creating two tags with the same title.
DROP INDEX idx_tags_title_unique;

CREATE UNIQUE INDEX idx_tags_title_unique ON tags(user_id, title) WHERE deleted = 0;

-- +goose Down
-- Down migration omitted (SQLite makes them painful and rollback is via backup).

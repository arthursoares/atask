-- +goose Up
CREATE UNIQUE INDEX idx_tags_title_unique ON tags (title) WHERE deleted = 0;

-- +goose Down
DROP INDEX IF EXISTS idx_tags_title_unique;

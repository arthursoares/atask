-- +goose Up
ALTER TABLE tasks ADD COLUMN time_slot TEXT;
ALTER TABLE sections ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sections ADD COLUMN collapsed INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tasks DROP COLUMN time_slot;
ALTER TABLE sections DROP COLUMN archived;
ALTER TABLE sections DROP COLUMN collapsed;

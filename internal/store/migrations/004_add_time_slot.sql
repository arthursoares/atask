-- +goose Up
ALTER TABLE tasks ADD COLUMN time_slot TEXT;

-- +goose Down
ALTER TABLE tasks DROP COLUMN time_slot;

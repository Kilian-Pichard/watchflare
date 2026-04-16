-- +goose Up
ALTER TABLE packages
  ADD COLUMN IF NOT EXISTS update_checked BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE packages DROP COLUMN IF EXISTS update_checked;

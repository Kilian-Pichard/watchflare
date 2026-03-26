-- +goose Up
-- Migration 009: Add username to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(50);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS username;

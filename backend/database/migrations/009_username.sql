-- Migration 009: Add username to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(50);

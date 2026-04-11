-- +goose Up
-- Migration 007: User preferences
-- Username and display preference columns.

ALTER TABLE users ADD COLUMN IF NOT EXISTS username              VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS time_format           VARCHAR(3)  DEFAULT '24h';
ALTER TABLE users ADD COLUMN IF NOT EXISTS temperature_unit      VARCHAR(15) DEFAULT 'celsius';
ALTER TABLE users ADD COLUMN IF NOT EXISTS network_unit          VARCHAR(5)  DEFAULT 'bytes';
ALTER TABLE users ADD COLUMN IF NOT EXISTS disk_unit             VARCHAR(5)  DEFAULT 'bytes';
ALTER TABLE users ADD COLUMN IF NOT EXISTS gauge_warning_threshold  INTEGER  DEFAULT 70;
ALTER TABLE users ADD COLUMN IF NOT EXISTS gauge_critical_threshold INTEGER  DEFAULT 90;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS gauge_critical_threshold;
ALTER TABLE users DROP COLUMN IF EXISTS gauge_warning_threshold;
ALTER TABLE users DROP COLUMN IF EXISTS disk_unit;
ALTER TABLE users DROP COLUMN IF EXISTS network_unit;
ALTER TABLE users DROP COLUMN IF EXISTS temperature_unit;
ALTER TABLE users DROP COLUMN IF EXISTS time_format;
ALTER TABLE users DROP COLUMN IF EXISTS username;

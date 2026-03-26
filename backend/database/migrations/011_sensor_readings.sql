-- +goose Up
-- Migration 011: Add sensor_readings JSONB column to metrics table
--
-- Stores all sensor readings as a snapshot alongside each metric row.
-- Format: [{key: "cpu_temp", temperature_celsius: 45.2}, ...]
--
-- Design note (hybrid approach):
--   sensor_readings (JSONB) — fast access to the latest snapshot, co-located
--     with the metric row, used by the real-time SSE feed and the metrics handler.
--   sensor_metrics table (migration 013) — normalized hypertable for per-sensor
--     historical queries and time-range aggregations.
--   Both are populated on each metric write; they serve different query patterns.
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS sensor_readings JSONB;

-- +goose Down
ALTER TABLE metrics DROP COLUMN IF EXISTS sensor_readings;

-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, create_hypertable, etc.) cannot run inside a transaction

-- +goose Up
-- =====================================================
-- Migration 013: Sensor Metrics Table
-- Normalized hypertable for per-sensor temperature data.
-- Uses time_bucket() at query time for all aggregations.
-- =====================================================

CREATE TABLE IF NOT EXISTS sensor_metrics (
    time        TIMESTAMPTZ NOT NULL,
    server_id   CHAR(36) NOT NULL,
    sensor_key  TEXT NOT NULL,
    temperature DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('sensor_metrics', 'time', if_not_exists => TRUE, migrate_data => TRUE);

-- 35 days retention: covers the 30d view
SELECT add_retention_policy('sensor_metrics', INTERVAL '35 days', if_not_exists => TRUE);

ALTER TABLE sensor_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id,sensor_key',
    timescaledb.compress_orderby = 'time DESC'
);
SELECT add_compression_policy('sensor_metrics', INTERVAL '1 day', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_sensor_metrics_server_time ON sensor_metrics (server_id, time DESC);

-- +goose Down
-- Not reversible without data loss

-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (create_hypertable, compression, policies) cannot run inside a transaction

-- +goose Up
-- Migration 006: Sensor metrics
-- Normalized hypertable for per-sensor temperature data.
-- Uses time_bucket() at query time for all aggregations.
--
-- Note: sensor_readings JSONB (snapshot, co-located with metrics rows) is
-- created in initMetricsTable() in db.go. This table stores normalized
-- per-sensor history for time-range queries and aggregations.

CREATE TABLE IF NOT EXISTS sensor_metrics (
    time        TIMESTAMPTZ      NOT NULL,
    host_id     CHAR(36)         NOT NULL,
    sensor_key  TEXT             NOT NULL,
    temperature DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('sensor_metrics', 'time', if_not_exists => TRUE, migrate_data => TRUE);

SELECT add_retention_policy('sensor_metrics', INTERVAL '35 days', if_not_exists => TRUE);

ALTER TABLE sensor_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'host_id,sensor_key',
    timescaledb.compress_orderby   = 'time DESC'
);
SELECT add_compression_policy('sensor_metrics', INTERVAL '1 day', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_sensor_metrics_host_time ON sensor_metrics (host_id, time DESC);

-- +goose Down
-- Not reversible without data loss

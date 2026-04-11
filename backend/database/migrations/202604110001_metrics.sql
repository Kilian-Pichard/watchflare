-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, compression, policies) cannot run inside a transaction

-- +goose Up
-- =====================================================
-- Migration 001: Metrics TimescaleDB Setup
-- Continuous aggregates, compression, retention, indexes.
-- Raw metrics table created by initMetricsTable() in db.go.
-- =====================================================

-- Retention policy for raw metrics (24h)
SELECT remove_retention_policy('metrics', if_exists => TRUE);
SELECT add_retention_policy('metrics', INTERVAL '24 hours', if_not_exists => TRUE);

-- =====================================================
-- Continuous aggregates
-- metrics_10min — used for 1h and 12h views (72 data points)
-- metrics_15min — used for 24h view (96 data points)
-- metrics_2h    — used for 7d view (84 data points)
-- metrics_8h    — used for 30d view (90 data points)
-- =====================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_10min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('10 minutes', timestamp) AS bucket,
    host_id,
    AVG(cpu_usage_percent)        AS cpu_usage_percent,
    AVG(memory_total_bytes)       AS memory_total_bytes,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_available_bytes)   AS memory_available_bytes,
    AVG(disk_total_bytes)         AS disk_total_bytes,
    AVG(disk_used_bytes)          AS disk_used_bytes,
    AVG(load_avg1_min)            AS load_avg1_min,
    AVG(load_avg5_min)            AS load_avg5_min,
    AVG(load_avg15_min)           AS load_avg15_min,
    AVG(uptime_seconds)           AS uptime_seconds,
    AVG(disk_read_bytes_per_sec)  AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius)  AS cpu_temperature_celsius,
    COUNT(*)                      AS data_points
FROM metrics
GROUP BY bucket, host_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_10min',
    start_offset      => INTERVAL '14 days',
    end_offset        => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('metrics_10min', INTERVAL '14 days', if_not_exists => TRUE);

CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_15min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('15 minutes', timestamp) AS bucket,
    host_id,
    AVG(cpu_usage_percent)        AS cpu_usage_percent,
    AVG(memory_total_bytes)       AS memory_total_bytes,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_available_bytes)   AS memory_available_bytes,
    AVG(disk_total_bytes)         AS disk_total_bytes,
    AVG(disk_used_bytes)          AS disk_used_bytes,
    AVG(load_avg1_min)            AS load_avg1_min,
    AVG(load_avg5_min)            AS load_avg5_min,
    AVG(load_avg15_min)           AS load_avg15_min,
    AVG(uptime_seconds)           AS uptime_seconds,
    AVG(disk_read_bytes_per_sec)  AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius)  AS cpu_temperature_celsius,
    COUNT(*)                      AS data_points
FROM metrics
GROUP BY bucket, host_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_15min',
    start_offset      => INTERVAL '30 days',
    end_offset        => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('metrics_15min', INTERVAL '30 days', if_not_exists => TRUE);

CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_2h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('2 hours', timestamp) AS bucket,
    host_id,
    AVG(cpu_usage_percent)        AS cpu_usage_percent,
    AVG(memory_total_bytes)       AS memory_total_bytes,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_available_bytes)   AS memory_available_bytes,
    AVG(disk_total_bytes)         AS disk_total_bytes,
    AVG(disk_used_bytes)          AS disk_used_bytes,
    AVG(load_avg1_min)            AS load_avg1_min,
    AVG(load_avg5_min)            AS load_avg5_min,
    AVG(load_avg15_min)           AS load_avg15_min,
    AVG(uptime_seconds)           AS uptime_seconds,
    AVG(disk_read_bytes_per_sec)  AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius)  AS cpu_temperature_celsius,
    COUNT(*)                      AS data_points
FROM metrics
GROUP BY bucket, host_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_2h',
    start_offset      => INTERVAL '90 days',
    end_offset        => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('metrics_2h', INTERVAL '90 days', if_not_exists => TRUE);

CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_8h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('8 hours', timestamp) AS bucket,
    host_id,
    AVG(cpu_usage_percent)        AS cpu_usage_percent,
    AVG(memory_total_bytes)       AS memory_total_bytes,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_available_bytes)   AS memory_available_bytes,
    AVG(disk_total_bytes)         AS disk_total_bytes,
    AVG(disk_used_bytes)          AS disk_used_bytes,
    AVG(load_avg1_min)            AS load_avg1_min,
    AVG(load_avg5_min)            AS load_avg5_min,
    AVG(load_avg15_min)           AS load_avg15_min,
    AVG(uptime_seconds)           AS uptime_seconds,
    AVG(disk_read_bytes_per_sec)  AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius)  AS cpu_temperature_celsius,
    COUNT(*)                      AS data_points
FROM metrics
GROUP BY bucket, host_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_8h',
    start_offset      => INTERVAL '1 year',
    end_offset        => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('metrics_8h', INTERVAL '1 year', if_not_exists => TRUE);

-- =====================================================
-- Compression: compress raw metrics older than 1 day
-- =====================================================
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'host_id',
    timescaledb.compress_orderby   = 'timestamp DESC'
);
SELECT add_compression_policy('metrics', INTERVAL '1 day', if_not_exists => TRUE);

-- =====================================================
-- Indexes
-- =====================================================
CREATE INDEX IF NOT EXISTS idx_metrics_host_time        ON metrics        (host_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_10min_host_bucket ON metrics_10min (host_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_15min_host_bucket ON metrics_15min (host_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_2h_host_bucket    ON metrics_2h    (host_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_8h_host_bucket    ON metrics_8h    (host_id, bucket DESC);

-- =====================================================
-- Initial refresh
-- =====================================================
CALL refresh_continuous_aggregate('metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_2h',    NULL, NULL);
CALL refresh_continuous_aggregate('metrics_8h',    NULL, NULL);

-- +goose Down
-- Not reversible without data loss

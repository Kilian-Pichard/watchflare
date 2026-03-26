-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, create_hypertable, etc.) cannot run inside a transaction

-- +goose Up
-- =====================================================
-- Migration 001: TimescaleDB Continuous Aggregates
-- =====================================================
-- Sets up continuous aggregates for all time range views.
-- Raw metrics retained for 24h, aggregates cover 10min–8h buckets.
-- =====================================================

-- 1. Set retention policy for raw metrics (24h)
SELECT remove_retention_policy('metrics', if_exists => TRUE);
SELECT add_retention_policy('metrics', INTERVAL '24 hours', if_not_exists => TRUE);

-- =====================================================
-- 2. CONTINUOUS AGGREGATE: 10 minutes
-- Used for the 12h view (72 data points)
-- =====================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_10min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('10 minutes', timestamp) AS bucket,
    server_id,
    AVG(cpu_usage_percent) AS cpu_usage_percent,
    AVG(memory_total_bytes) AS memory_total_bytes,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_available_bytes) AS memory_available_bytes,
    AVG(disk_total_bytes) AS disk_total_bytes,
    AVG(disk_used_bytes) AS disk_used_bytes,
    AVG(load_avg1_min) AS load_avg1_min,
    AVG(load_avg5_min) AS load_avg5_min,
    AVG(load_avg15_min) AS load_avg15_min,
    AVG(uptime_seconds) AS uptime_seconds,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

-- Refresh every 10 minutes
SELECT add_continuous_aggregate_policy('metrics_10min',
    start_offset => INTERVAL '14 days',
    end_offset => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists => TRUE
);

-- Retention: keep 14 days of 10min aggregates
SELECT add_retention_policy('metrics_10min',
    INTERVAL '14 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 3. CONTINUOUS AGGREGATE: 15 minutes
-- Used for the 24h view (96 data points)
-- =====================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_15min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('15 minutes', timestamp) AS bucket,
    server_id,
    AVG(cpu_usage_percent) AS cpu_usage_percent,
    AVG(memory_total_bytes) AS memory_total_bytes,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_available_bytes) AS memory_available_bytes,
    AVG(disk_total_bytes) AS disk_total_bytes,
    AVG(disk_used_bytes) AS disk_used_bytes,
    AVG(load_avg1_min) AS load_avg1_min,
    AVG(load_avg5_min) AS load_avg5_min,
    AVG(load_avg15_min) AS load_avg15_min,
    AVG(uptime_seconds) AS uptime_seconds,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

-- Refresh every 15 minutes
SELECT add_continuous_aggregate_policy('metrics_15min',
    start_offset => INTERVAL '30 days',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE
);

-- Retention: keep 30 days of 15min aggregates
SELECT add_retention_policy('metrics_15min',
    INTERVAL '30 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 4. CONTINUOUS AGGREGATE: 2 hours
-- Used for the 7d view (84 data points)
-- =====================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_2h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('2 hours', timestamp) AS bucket,
    server_id,
    AVG(cpu_usage_percent) AS cpu_usage_percent,
    AVG(memory_total_bytes) AS memory_total_bytes,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_available_bytes) AS memory_available_bytes,
    AVG(disk_total_bytes) AS disk_total_bytes,
    AVG(disk_used_bytes) AS disk_used_bytes,
    AVG(load_avg1_min) AS load_avg1_min,
    AVG(load_avg5_min) AS load_avg5_min,
    AVG(load_avg15_min) AS load_avg15_min,
    AVG(uptime_seconds) AS uptime_seconds,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

-- Refresh every 2 hours
SELECT add_continuous_aggregate_policy('metrics_2h',
    start_offset => INTERVAL '90 days',
    end_offset => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists => TRUE
);

-- Retention: keep 90 days of 2h aggregates
SELECT add_retention_policy('metrics_2h',
    INTERVAL '90 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 5. CONTINUOUS AGGREGATE: 8 hours
-- Used for the 30d view (90 data points)
-- =====================================================
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_8h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('8 hours', timestamp) AS bucket,
    server_id,
    AVG(cpu_usage_percent) AS cpu_usage_percent,
    AVG(memory_total_bytes) AS memory_total_bytes,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_available_bytes) AS memory_available_bytes,
    AVG(disk_total_bytes) AS disk_total_bytes,
    AVG(disk_used_bytes) AS disk_used_bytes,
    AVG(load_avg1_min) AS load_avg1_min,
    AVG(load_avg5_min) AS load_avg5_min,
    AVG(load_avg15_min) AS load_avg15_min,
    AVG(uptime_seconds) AS uptime_seconds,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

-- Refresh every 8 hours
SELECT add_continuous_aggregate_policy('metrics_8h',
    start_offset => INTERVAL '1 year',
    end_offset => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists => TRUE
);

-- Retention: keep 1 year of 8h aggregates
SELECT add_retention_policy('metrics_8h',
    INTERVAL '1 year',
    if_not_exists => TRUE
);

-- =====================================================
-- 6. COMPRESSION
-- Compress raw metrics older than 1 day
-- =====================================================
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('metrics',
    INTERVAL '1 day',
    if_not_exists => TRUE
);

-- =====================================================
-- 7. INDEXES
-- =====================================================
CREATE INDEX IF NOT EXISTS idx_metrics_server_time ON metrics (server_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_10min_server_bucket ON metrics_10min (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_15min_server_bucket ON metrics_15min (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_2h_server_bucket ON metrics_2h (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_8h_server_bucket ON metrics_8h (server_id, bucket DESC);

-- =====================================================
-- 8. INITIAL REFRESH
-- =====================================================
CALL refresh_continuous_aggregate('metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_2h', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_8h', NULL, NULL);

-- =====================================================
-- Summary:
-- Raw metrics (metrics):       24h retention, compressed after 1d
-- 10min (metrics_10min):       14d retention, used for 1h/12h views
-- 15min (metrics_15min):       30d retention, used for 24h view
-- 2h   (metrics_2h):           90d retention, used for 7d view
-- 8h   (metrics_8h):           1y  retention, used for 30d view
-- =====================================================

-- +goose Down
-- Not reversible without data loss

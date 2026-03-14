-- =====================================================
-- Migration 006: Add Disk I/O, Network, Temperature metrics
-- =====================================================

-- 1. Add new columns to metrics hypertable
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_read_bytes_per_sec BIGINT DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS disk_write_bytes_per_sec BIGINT DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_rx_bytes_per_sec BIGINT DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS network_tx_bytes_per_sec BIGINT DEFAULT 0;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_temperature_celsius DOUBLE PRECISION DEFAULT 0;

-- =====================================================
-- 2. Recreate continuous aggregates with new columns
-- (Cannot ALTER a materialized view, must DROP + CREATE)
-- =====================================================

-- 2a. metrics_10min (12h view)
DROP MATERIALIZED VIEW IF EXISTS metrics_10min CASCADE;

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
    AVG(disk_read_bytes_per_sec) AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius) AS cpu_temperature_celsius,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_10min',
    start_offset => INTERVAL '14 days',
    end_offset => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists => TRUE
);

SELECT add_retention_policy('metrics_10min',
    INTERVAL '14 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_metrics_10min_server_bucket ON metrics_10min (server_id, bucket DESC);

-- 2b. metrics_15min (24h view)
DROP MATERIALIZED VIEW IF EXISTS metrics_15min CASCADE;

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
    AVG(disk_read_bytes_per_sec) AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius) AS cpu_temperature_celsius,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_15min',
    start_offset => INTERVAL '30 days',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE
);

SELECT add_retention_policy('metrics_15min',
    INTERVAL '30 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_metrics_15min_server_bucket ON metrics_15min (server_id, bucket DESC);

-- 2c. metrics_2h (7d view)
DROP MATERIALIZED VIEW IF EXISTS metrics_2h CASCADE;

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
    AVG(disk_read_bytes_per_sec) AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius) AS cpu_temperature_celsius,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_2h',
    start_offset => INTERVAL '90 days',
    end_offset => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists => TRUE
);

SELECT add_retention_policy('metrics_2h',
    INTERVAL '90 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_metrics_2h_server_bucket ON metrics_2h (server_id, bucket DESC);

-- 2d. metrics_8h (30d view)
DROP MATERIALIZED VIEW IF EXISTS metrics_8h CASCADE;

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
    AVG(disk_read_bytes_per_sec) AS disk_read_bytes_per_sec,
    AVG(disk_write_bytes_per_sec) AS disk_write_bytes_per_sec,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    AVG(cpu_temperature_celsius) AS cpu_temperature_celsius,
    COUNT(*) AS data_points
FROM metrics
GROUP BY bucket, server_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('metrics_8h',
    start_offset => INTERVAL '1 year',
    end_offset => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists => TRUE
);

SELECT add_retention_policy('metrics_8h',
    INTERVAL '1 year',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_metrics_8h_server_bucket ON metrics_8h (server_id, bucket DESC);

-- =====================================================
-- 3. Refresh continuous aggregates with existing data
-- =====================================================
CALL refresh_continuous_aggregate('metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_2h', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_8h', NULL, NULL);

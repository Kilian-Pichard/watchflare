-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, create_hypertable, etc.) cannot run inside a transaction

-- +goose Up
-- =====================================================
-- Migration 008: Continuous aggregates for container metrics
-- Same bucket pattern as system metrics (10min, 15min, 2h, 8h)
-- =====================================================

-- 1. container_metrics_10min (12h view)
CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_10min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('10 minutes', timestamp) AS bucket,
    server_id,
    container_id,
    container_name,
    AVG(cpu_percent) AS cpu_percent,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_limit_bytes) AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*) AS data_points
FROM container_metrics
GROUP BY bucket, server_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_10min',
    start_offset => INTERVAL '14 days',
    end_offset => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists => TRUE
);

SELECT add_retention_policy('container_metrics_10min',
    INTERVAL '14 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_container_metrics_10min_server_bucket
    ON container_metrics_10min (server_id, bucket DESC);

-- 2. container_metrics_15min (24h view)
CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_15min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('15 minutes', timestamp) AS bucket,
    server_id,
    container_id,
    container_name,
    AVG(cpu_percent) AS cpu_percent,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_limit_bytes) AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*) AS data_points
FROM container_metrics
GROUP BY bucket, server_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_15min',
    start_offset => INTERVAL '30 days',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE
);

SELECT add_retention_policy('container_metrics_15min',
    INTERVAL '30 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_container_metrics_15min_server_bucket
    ON container_metrics_15min (server_id, bucket DESC);

-- 3. container_metrics_2h (7d view)
CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_2h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('2 hours', timestamp) AS bucket,
    server_id,
    container_id,
    container_name,
    AVG(cpu_percent) AS cpu_percent,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_limit_bytes) AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*) AS data_points
FROM container_metrics
GROUP BY bucket, server_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_2h',
    start_offset => INTERVAL '90 days',
    end_offset => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists => TRUE
);

SELECT add_retention_policy('container_metrics_2h',
    INTERVAL '90 days',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_container_metrics_2h_server_bucket
    ON container_metrics_2h (server_id, bucket DESC);

-- 4. container_metrics_8h (30d view)
CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_8h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('8 hours', timestamp) AS bucket,
    server_id,
    container_id,
    container_name,
    AVG(cpu_percent) AS cpu_percent,
    AVG(memory_used_bytes) AS memory_used_bytes,
    AVG(memory_limit_bytes) AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*) AS data_points
FROM container_metrics
GROUP BY bucket, server_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_8h',
    start_offset => INTERVAL '1 year',
    end_offset => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists => TRUE
);

SELECT add_retention_policy('container_metrics_8h',
    INTERVAL '1 year',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_container_metrics_8h_server_bucket
    ON container_metrics_8h (server_id, bucket DESC);

-- =====================================================
-- 5. Refresh continuous aggregates with existing data
-- =====================================================
CALL refresh_continuous_aggregate('container_metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_2h', NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_8h', NULL, NULL);

-- +goose Down
-- Not reversible without data loss

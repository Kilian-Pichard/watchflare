-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (create_hypertable, CREATE MATERIALIZED VIEW, policies) cannot run inside a transaction

-- +goose Up
-- Migration 005: Container metrics
-- Per-container Docker metrics hypertable and continuous aggregates.

CREATE TABLE IF NOT EXISTS container_metrics (
    id                       CHAR(36)          NOT NULL,
    timestamp                TIMESTAMPTZ       NOT NULL,
    host_id                  CHAR(36)          NOT NULL,
    container_id             TEXT              NOT NULL,
    container_name           TEXT              NOT NULL,
    image                    TEXT              NOT NULL DEFAULT '',
    cpu_percent              DOUBLE PRECISION  NOT NULL DEFAULT 0,
    memory_used_bytes        BIGINT            NOT NULL DEFAULT 0,
    memory_limit_bytes       BIGINT            NOT NULL DEFAULT 0,
    network_rx_bytes_per_sec BIGINT            NOT NULL DEFAULT 0,
    network_tx_bytes_per_sec BIGINT            NOT NULL DEFAULT 0,
    PRIMARY KEY (id, timestamp)
);

SELECT create_hypertable('container_metrics', 'timestamp', if_not_exists => TRUE);

ALTER TABLE container_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'host_id,container_id'
);
SELECT add_compression_policy('container_metrics', INTERVAL '2 days',  if_not_exists => TRUE);
SELECT add_retention_policy('container_metrics',   INTERVAL '30 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_container_metrics_host_time ON container_metrics (host_id, timestamp DESC);

-- =====================================================
-- Continuous aggregates (same bucket pattern as system metrics)
-- 10min — 1h/12h views  |  15min — 24h view
-- 2h    — 7d view       |  8h    — 30d view
-- =====================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_10min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('10 minutes', timestamp) AS bucket,
    host_id,
    container_id,
    container_name,
    AVG(cpu_percent)              AS cpu_percent,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_limit_bytes)       AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*)                      AS data_points
FROM container_metrics
GROUP BY bucket, host_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_10min',
    start_offset      => INTERVAL '14 days',
    end_offset        => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('container_metrics_10min', INTERVAL '14 days', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_container_metrics_10min_host_bucket ON container_metrics_10min (host_id, bucket DESC);

CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_15min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('15 minutes', timestamp) AS bucket,
    host_id,
    container_id,
    container_name,
    AVG(cpu_percent)              AS cpu_percent,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_limit_bytes)       AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*)                      AS data_points
FROM container_metrics
GROUP BY bucket, host_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_15min',
    start_offset      => INTERVAL '30 days',
    end_offset        => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('container_metrics_15min', INTERVAL '30 days', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_container_metrics_15min_host_bucket ON container_metrics_15min (host_id, bucket DESC);

CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_2h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('2 hours', timestamp) AS bucket,
    host_id,
    container_id,
    container_name,
    AVG(cpu_percent)              AS cpu_percent,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_limit_bytes)       AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*)                      AS data_points
FROM container_metrics
GROUP BY bucket, host_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_2h',
    start_offset      => INTERVAL '90 days',
    end_offset        => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('container_metrics_2h', INTERVAL '90 days', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_container_metrics_2h_host_bucket ON container_metrics_2h (host_id, bucket DESC);

CREATE MATERIALIZED VIEW IF NOT EXISTS container_metrics_8h
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('8 hours', timestamp) AS bucket,
    host_id,
    container_id,
    container_name,
    AVG(cpu_percent)              AS cpu_percent,
    AVG(memory_used_bytes)        AS memory_used_bytes,
    AVG(memory_limit_bytes)       AS memory_limit_bytes,
    AVG(network_rx_bytes_per_sec) AS network_rx_bytes_per_sec,
    AVG(network_tx_bytes_per_sec) AS network_tx_bytes_per_sec,
    COUNT(*)                      AS data_points
FROM container_metrics
GROUP BY bucket, host_id, container_id, container_name
WITH NO DATA;

SELECT add_continuous_aggregate_policy('container_metrics_8h',
    start_offset      => INTERVAL '1 year',
    end_offset        => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists     => TRUE
);
SELECT add_retention_policy('container_metrics_8h', INTERVAL '1 year', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_container_metrics_8h_host_bucket ON container_metrics_8h (host_id, bucket DESC);

CALL refresh_continuous_aggregate('container_metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_2h',    NULL, NULL);
CALL refresh_continuous_aggregate('container_metrics_8h',    NULL, NULL);

-- +goose Down
-- Not reversible without data loss

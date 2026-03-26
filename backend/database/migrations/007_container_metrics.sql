-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, create_hypertable, etc.) cannot run inside a transaction

-- +goose Up
-- Migration 007: Container metrics table
-- Stores per-container Docker metrics (CPU, memory, network) as a TimescaleDB hypertable

CREATE TABLE IF NOT EXISTS container_metrics (
    id CHAR(36) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    server_id CHAR(36) NOT NULL,
    container_id TEXT NOT NULL,
    container_name TEXT NOT NULL,
    image TEXT NOT NULL DEFAULT '',
    cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_used_bytes BIGINT NOT NULL DEFAULT 0,
    memory_limit_bytes BIGINT NOT NULL DEFAULT 0,
    network_rx_bytes_per_sec BIGINT NOT NULL DEFAULT 0,
    network_tx_bytes_per_sec BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id, timestamp)
);

SELECT create_hypertable('container_metrics', 'timestamp', if_not_exists => TRUE);

ALTER TABLE container_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id, container_id'
);

SELECT add_compression_policy('container_metrics', INTERVAL '2 days', if_not_exists => TRUE);

SELECT add_retention_policy('container_metrics', INTERVAL '30 days', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_container_metrics_server_time
    ON container_metrics (server_id, timestamp DESC);

-- +goose Down
-- Not reversible without data loss

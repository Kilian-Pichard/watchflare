-- +goose NO TRANSACTION
-- Required: TimescaleDB DDL (CREATE MATERIALIZED VIEW, create_hypertable, etc.) cannot run inside a transaction

-- +goose Up
-- Migration 003: Package Inventory Tables
-- Creates tables for package inventory collection and history

-- =====================================================
-- Current Package State (one row per package per server)
-- =====================================================
CREATE TABLE IF NOT EXISTS packages (
    id BIGSERIAL PRIMARY KEY,
    server_id CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,

    -- Package identification
    name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    architecture VARCHAR(50),
    package_manager VARCHAR(20) NOT NULL, -- 'dpkg', 'rpm', 'brew', etc.

    -- Package metadata
    source VARCHAR(255),          -- Repository or source
    installed_at TIMESTAMPTZ,     -- When package was installed (if available)
    package_size BIGINT,          -- Size in bytes
    description VARCHAR(100),     -- Short description (max 100 chars)

    -- State tracking
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite unique constraint: one version per package per server
    CONSTRAINT packages_server_name_unique UNIQUE (server_id, name, package_manager)
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_packages_server ON packages(server_id);
CREATE INDEX IF NOT EXISTS idx_packages_name ON packages(name);
CREATE INDEX IF NOT EXISTS idx_packages_name_version ON packages(name, version);
CREATE INDEX IF NOT EXISTS idx_packages_last_seen ON packages(last_seen DESC);

-- =====================================================
-- Package History (TimescaleDB Hypertable - snapshots)
-- =====================================================
CREATE TABLE IF NOT EXISTS package_history (
    id BIGSERIAL,
    timestamp TIMESTAMPTZ NOT NULL,
    server_id CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,

    -- Package data (snapshot at this time)
    name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    architecture VARCHAR(50),
    package_manager VARCHAR(20) NOT NULL,
    source VARCHAR(255),
    package_size BIGINT,
    description VARCHAR(100),

    -- Change type
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('added', 'removed', 'updated', 'initial')),

    PRIMARY KEY (id, timestamp)
);

-- Convert to hypertable (partitioned by time)
SELECT create_hypertable('package_history', 'timestamp', if_not_exists => TRUE);

-- Retention policy: keep package history for 365 days
SELECT add_retention_policy('package_history', INTERVAL '365 days', if_not_exists => TRUE);

-- Compression: compress data older than 7 days
ALTER TABLE package_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('package_history', INTERVAL '7 days', if_not_exists => TRUE);

-- Indexes for historical queries
CREATE INDEX IF NOT EXISTS idx_package_history_server_time ON package_history(server_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_package_history_name ON package_history(name, timestamp DESC);

-- =====================================================
-- Package Collection Metadata (tracking collection jobs)
-- =====================================================
CREATE TABLE IF NOT EXISTS package_collections (
    id BIGSERIAL PRIMARY KEY,
    server_id CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Collection details
    collection_type VARCHAR(20) NOT NULL CHECK (collection_type IN ('full', 'delta', 'initial')),
    package_count INTEGER NOT NULL,
    changes_count INTEGER DEFAULT 0,     -- Number of changes detected
    duration_ms INTEGER,                 -- Collection time in milliseconds

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed', 'partial')),
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_package_collections_server_time ON package_collections(server_id, timestamp DESC);

-- +goose Down
-- Not reversible without data loss

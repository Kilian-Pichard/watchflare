-- +goose Up
-- Migration: Add dropped_metrics table for tracking lost metrics
-- This table stores reports from agents when metrics are dropped after max retries

CREATE TABLE IF NOT EXISTS dropped_metrics (
    id BIGSERIAL PRIMARY KEY,
    agent_id CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    count INTEGER NOT NULL,
    first_dropped_at TIMESTAMPTZ NOT NULL,
    last_dropped_at TIMESTAMPTZ NOT NULL,
    reason VARCHAR(100) NOT NULL,
    reported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Index for fast lookups by agent and time
    CONSTRAINT dropped_metrics_count_positive CHECK (count > 0)
);

CREATE INDEX IF NOT EXISTS idx_dropped_metrics_agent_time ON dropped_metrics(agent_id, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_dropped_metrics_reported_at ON dropped_metrics(reported_at DESC);

-- View: Aggregate dropped metrics summary for the last 24 hours
CREATE OR REPLACE VIEW agent_dropped_metrics_summary AS
SELECT
    s.id AS agent_id,
    s.name AS hostname,
    COALESCE(SUM(dm.count), 0) AS total_dropped,
    MIN(dm.first_dropped_at) AS first_dropped_at,
    MAX(dm.last_dropped_at) AS last_dropped_at,
    MAX(dm.reported_at) AS last_reported_at
FROM servers s
LEFT JOIN dropped_metrics dm ON dm.agent_id = s.id
WHERE dm.reported_at > NOW() - INTERVAL '24 hours' OR dm.reported_at IS NULL
GROUP BY s.id, s.name
HAVING SUM(dm.count) > 0
ORDER BY total_dropped DESC;

COMMENT ON TABLE dropped_metrics IS 'Records of metrics that were dropped by agents after max retries';
COMMENT ON VIEW agent_dropped_metrics_summary IS 'Summary of dropped metrics per agent in the last 24 hours';

-- +goose Down
DROP VIEW IF EXISTS agent_dropped_metrics_summary;
DROP INDEX IF EXISTS idx_dropped_metrics_reported_at;
DROP INDEX IF EXISTS idx_dropped_metrics_agent_time;
DROP TABLE IF EXISTS dropped_metrics;

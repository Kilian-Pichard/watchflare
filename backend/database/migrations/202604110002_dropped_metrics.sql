-- +goose Up
-- Migration: Add dropped_metrics table for tracking lost metrics
-- This table stores reports from agents when metrics are dropped after max retries

CREATE TABLE IF NOT EXISTS dropped_metrics (
    id               BIGSERIAL   PRIMARY KEY,
    host_id          CHAR(36)    NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    count            INTEGER     NOT NULL,
    first_dropped_at TIMESTAMPTZ NOT NULL,
    last_dropped_at  TIMESTAMPTZ NOT NULL,
    reason           VARCHAR(100) NOT NULL,
    reported_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dropped_metrics_count_positive CHECK (count > 0)
);

CREATE INDEX IF NOT EXISTS idx_dropped_metrics_host_time  ON dropped_metrics(host_id, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_dropped_metrics_reported_at ON dropped_metrics(reported_at DESC);

-- View: Dropped metrics summary per host for the last 24 hours
CREATE OR REPLACE VIEW agent_dropped_metrics_summary AS
SELECT
    h.id   AS host_id,
    h.name AS hostname,
    SUM(dm.count)            AS total_dropped,
    MIN(dm.first_dropped_at) AS first_dropped_at,
    MAX(dm.last_dropped_at)  AS last_dropped_at,
    MAX(dm.reported_at)      AS last_reported_at
FROM hosts h
JOIN dropped_metrics dm ON dm.host_id = h.id
WHERE dm.reported_at > NOW() - INTERVAL '24 hours'
GROUP BY h.id, h.name
HAVING SUM(dm.count) > 0
ORDER BY total_dropped DESC;

COMMENT ON TABLE dropped_metrics IS 'Records of metrics that were dropped by agents after max retries';
COMMENT ON VIEW agent_dropped_metrics_summary IS 'Summary of dropped metrics per agent in the last 24 hours';

-- +goose Down
DROP VIEW IF EXISTS agent_dropped_metrics_summary;
DROP INDEX IF EXISTS idx_dropped_metrics_reported_at;
DROP INDEX IF EXISTS idx_dropped_metrics_host_time;
DROP TABLE IF EXISTS dropped_metrics;

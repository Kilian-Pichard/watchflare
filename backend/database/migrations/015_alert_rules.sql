-- +goose Up
-- Global alert rules (one row per metric type, seeded with defaults)
CREATE TABLE alert_rules (
	metric_type      VARCHAR(20) PRIMARY KEY
		CHECK (metric_type IN ('server_down','cpu_usage','memory_usage','disk_usage','load_avg','load_avg_5','load_avg_15','temperature')),
	enabled          BOOLEAN NOT NULL DEFAULT FALSE,
	threshold        DOUBLE PRECISION NOT NULL DEFAULT 0,
	duration_minutes INTEGER NOT NULL DEFAULT 5
		CHECK (duration_minutes >= 1),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO alert_rules (metric_type, enabled, threshold, duration_minutes) VALUES
	('server_down',   FALSE, 0,  1),
	('cpu_usage',     FALSE, 90, 5),
	('memory_usage',  FALSE, 90, 5),
	('disk_usage',    FALSE, 90, 5),
	('load_avg',      FALSE, 5,  5),
	('load_avg_5',    FALSE, 5,  5),
	('load_avg_15',   FALSE, 5,  5),
	('temperature',   FALSE, 80, 5)
ON CONFLICT DO NOTHING;

-- Per-server overrides (optional, server-level rules take precedence over global)
CREATE TABLE server_alert_rules (
	server_id        CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	metric_type      VARCHAR(20) NOT NULL
		CHECK (metric_type IN ('server_down','cpu_usage','memory_usage','disk_usage','load_avg','load_avg_5','load_avg_15','temperature')),
	enabled          BOOLEAN NOT NULL DEFAULT FALSE,
	threshold        DOUBLE PRECISION NOT NULL DEFAULT 0,
	duration_minutes INTEGER NOT NULL DEFAULT 5
		CHECK (duration_minutes >= 1),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (server_id, metric_type)
);

-- Alert incidents (one row per firing alert, resolved when condition clears)
CREATE TABLE alert_incidents (
	id              CHAR(36) PRIMARY KEY,
	server_id       CHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
	metric_type     VARCHAR(20) NOT NULL,
	started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at     TIMESTAMPTZ,
	notified        BOOLEAN NOT NULL DEFAULT FALSE,
	threshold_value DOUBLE PRECISION NOT NULL DEFAULT 0,
	current_value   DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE INDEX idx_alert_incidents_server ON alert_incidents(server_id);
CREATE INDEX idx_alert_incidents_open ON alert_incidents(server_id, metric_type) WHERE resolved_at IS NULL;


-- +goose Down
DROP TABLE IF EXISTS alert_incidents;
DROP TABLE IF EXISTS server_alert_rules;
DROP TABLE IF EXISTS alert_rules;

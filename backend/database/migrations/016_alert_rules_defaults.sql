-- +goose Up
-- Enable server_down alert by default (most universally useful alert)
-- Adjust thresholds: disk 85% (more conservative), load averages 2.0 (sane default)
UPDATE alert_rules SET enabled = TRUE                    WHERE metric_type = 'server_down';
UPDATE alert_rules SET threshold = 85                   WHERE metric_type = 'disk_usage';
UPDATE alert_rules SET threshold = 2.0, duration_minutes = 5 WHERE metric_type IN ('load_avg', 'load_avg_5', 'load_avg_15');

-- +goose Down
UPDATE alert_rules SET enabled = FALSE   WHERE metric_type = 'server_down';
UPDATE alert_rules SET threshold = 90    WHERE metric_type = 'disk_usage';
UPDATE alert_rules SET threshold = 5.0   WHERE metric_type IN ('load_avg', 'load_avg_5', 'load_avg_15');

-- =====================================================
-- TimescaleDB Continuous Aggregates Migration
-- =====================================================
-- Ce fichier configure les agrégations continues pour
-- optimiser les requêtes sur différentes granularités
-- =====================================================

-- 1. Modifier la politique de rétention (30j → 24h pour données brutes)
SELECT remove_retention_policy('metrics', if_exists => TRUE);
SELECT add_retention_policy('metrics', INTERVAL '24 hours', if_not_exists => TRUE);

-- =====================================================
-- 2. CONTINUOUS AGGREGATE: 10 minutes
-- Utilisé pour la vue 12h (72 points)
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

-- Refresh policy: rafraîchir toutes les 10 minutes
SELECT add_continuous_aggregate_policy('metrics_10min',
    start_offset => INTERVAL '14 days',
    end_offset => INTERVAL '10 minutes',
    schedule_interval => INTERVAL '10 minutes',
    if_not_exists => TRUE
);

-- Rétention: garder 14 jours d'agrégats 10min
SELECT add_retention_policy('metrics_10min',
    INTERVAL '14 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 3. CONTINUOUS AGGREGATE: 15 minutes
-- Utilisé pour la vue 24h (96 points)
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

-- Refresh policy: rafraîchir toutes les 15 minutes
SELECT add_continuous_aggregate_policy('metrics_15min',
    start_offset => INTERVAL '30 days',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE
);

-- Rétention: garder 30 jours d'agrégats 15min
SELECT add_retention_policy('metrics_15min',
    INTERVAL '30 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 4. CONTINUOUS AGGREGATE: 2 heures
-- Utilisé pour la vue 7 jours (84 points)
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

-- Refresh policy: rafraîchir toutes les 2 heures
SELECT add_continuous_aggregate_policy('metrics_2h',
    start_offset => INTERVAL '90 days',
    end_offset => INTERVAL '2 hours',
    schedule_interval => INTERVAL '2 hours',
    if_not_exists => TRUE
);

-- Rétention: garder 90 jours d'agrégats 2h
SELECT add_retention_policy('metrics_2h',
    INTERVAL '90 days',
    if_not_exists => TRUE
);

-- =====================================================
-- 5. CONTINUOUS AGGREGATE: 8 heures
-- Utilisé pour la vue 30 jours (90 points)
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

-- Refresh policy: rafraîchir toutes les 8 heures
SELECT add_continuous_aggregate_policy('metrics_8h',
    start_offset => INTERVAL '1 year',
    end_offset => INTERVAL '8 hours',
    schedule_interval => INTERVAL '8 hours',
    if_not_exists => TRUE
);

-- Rétention: garder 1 an d'agrégats 8h
SELECT add_retention_policy('metrics_8h',
    INTERVAL '1 year',
    if_not_exists => TRUE
);

-- =====================================================
-- 6. COMPRESSION
-- Active la compression sur la hypertable pour économiser l'espace
-- =====================================================
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

-- Compression policy: compresser les données > 1 jour
SELECT add_compression_policy('metrics',
    INTERVAL '1 day',
    if_not_exists => TRUE
);

-- =====================================================
-- 7. INDEX OPTIMISÉS
-- =====================================================
-- Index composé pour les requêtes fréquentes
CREATE INDEX IF NOT EXISTS idx_metrics_server_time ON metrics (server_id, timestamp DESC);

-- Index pour les continuous aggregates
CREATE INDEX IF NOT EXISTS idx_metrics_10min_server_bucket ON metrics_10min (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_15min_server_bucket ON metrics_15min (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_2h_server_bucket ON metrics_2h (server_id, bucket DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_8h_server_bucket ON metrics_8h (server_id, bucket DESC);

-- =====================================================
-- 8. REFRESH INITIAL DES CONTINUOUS AGGREGATES
-- =====================================================
-- Rafraîchir les vues avec les données existantes
CALL refresh_continuous_aggregate('metrics_10min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_15min', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_2h', NULL, NULL);
CALL refresh_continuous_aggregate('metrics_8h', NULL, NULL);

-- =====================================================
-- RÉSUMÉ DE LA CONFIGURATION
-- =====================================================
-- Données brutes (metrics):           Rétention 24h, Compression après 1j
-- Agrégats 10min (metrics_10min):     Rétention 14j, Utilisé pour 12h
-- Agrégats 15min (metrics_15min):     Rétention 30j, Utilisé pour 24h
-- Agrégats 2h (metrics_2h):           Rétention 90j, Utilisé pour 7j
-- Agrégats 8h (metrics_8h):           Rétention 1an, Utilisé pour 30j
-- =====================================================

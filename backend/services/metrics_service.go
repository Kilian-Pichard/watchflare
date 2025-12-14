package services

import (
	"errors"
	"fmt"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"gorm.io/gorm"
)

// MetricsQueryParams represents the query parameters for metrics retrieval
type MetricsQueryParams struct {
	ServerID string
	Start    time.Time
	End      time.Time
	Interval string // e.g., "1m", "5m", "15m", "1h"
}

// MetricDataPoint represents an aggregated metric data point
type MetricDataPoint struct {
	Timestamp            time.Time `json:"timestamp"`
	CPUUsagePercent      float64   `json:"cpu_usage_percent"`
	MemoryTotalBytes     uint64    `json:"memory_total_bytes"`
	MemoryUsedBytes      uint64    `json:"memory_used_bytes"`
	MemoryAvailableBytes uint64    `json:"memory_available_bytes"`
	LoadAvg1Min          float64   `json:"load_avg_1min"`
	LoadAvg5Min          float64   `json:"load_avg_5min"`
	LoadAvg15Min         float64   `json:"load_avg_15min"`
	DiskTotalBytes       uint64    `json:"disk_total_bytes"`
	DiskUsedBytes        uint64    `json:"disk_used_bytes"`
	UptimeSeconds        uint64    `json:"uptime_seconds"`
}

// GetMetrics retrieves metrics for a server with optional time range and aggregation
func GetMetrics(params MetricsQueryParams) ([]MetricDataPoint, error) {
	// Verify server exists
	var server models.Server
	if err := database.DB.Where("id = ?", params.ServerID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("server not found")
		}
		return nil, err
	}

	var results []MetricDataPoint

	// If no interval specified, return raw data
	if params.Interval == "" {
		var metrics []models.Metric
		query := database.DB.Where("server_id = ? AND timestamp >= ? AND timestamp <= ?",
			params.ServerID, params.Start, params.End).
			Order("timestamp ASC")

		if err := query.Find(&metrics).Error; err != nil {
			return nil, err
		}

		// Convert to response format
		for _, m := range metrics {
			results = append(results, MetricDataPoint{
				Timestamp:            m.Timestamp,
				CPUUsagePercent:      m.CPUUsagePercent,
				MemoryTotalBytes:     m.MemoryTotalBytes,
				MemoryUsedBytes:      m.MemoryUsedBytes,
				MemoryAvailableBytes: m.MemoryAvailableBytes,
				LoadAvg1Min:          m.LoadAvg1Min,
				LoadAvg5Min:          m.LoadAvg5Min,
				LoadAvg15Min:         m.LoadAvg15Min,
				DiskTotalBytes:       m.DiskTotalBytes,
				DiskUsedBytes:        m.DiskUsedBytes,
				UptimeSeconds:        m.UptimeSeconds,
			})
		}

		return results, nil
	}

	// Use TimescaleDB time_bucket for aggregation
	bucketInterval, err := parseInterval(params.Interval)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT
			time_bucket($1, timestamp) AS timestamp,
			AVG(cpu_usage_percent) AS cpu_usage_percent,
			CAST(AVG(memory_total_bytes) AS BIGINT) AS memory_total_bytes,
			CAST(AVG(memory_used_bytes) AS BIGINT) AS memory_used_bytes,
			CAST(AVG(memory_available_bytes) AS BIGINT) AS memory_available_bytes,
			AVG(load_avg1_min) AS load_avg_1min,
			AVG(load_avg5_min) AS load_avg_5min,
			AVG(load_avg15_min) AS load_avg_15min,
			CAST(AVG(disk_total_bytes) AS BIGINT) AS disk_total_bytes,
			CAST(AVG(disk_used_bytes) AS BIGINT) AS disk_used_bytes,
			CAST(AVG(uptime_seconds) AS BIGINT) AS uptime_seconds
		FROM metrics
		WHERE server_id = $2 AND timestamp >= $3 AND timestamp <= $4
		GROUP BY time_bucket($1, timestamp)
		ORDER BY timestamp ASC
	`

	if err := database.DB.Raw(query, bucketInterval, params.ServerID, params.Start, params.End).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// parseInterval converts interval string (e.g., "1m", "5m", "1h") to PostgreSQL interval format
func parseInterval(interval string) (string, error) {
	validIntervals := map[string]string{
		"1m":  "1 minute",
		"5m":  "5 minutes",
		"15m": "15 minutes",
		"30m": "30 minutes",
		"1h":  "1 hour",
		"6h":  "6 hours",
		"12h": "12 hours",
		"1d":  "1 day",
	}

	pgInterval, ok := validIntervals[interval]
	if !ok {
		return "", fmt.Errorf("invalid interval: %s. Valid intervals: 1m, 5m, 15m, 30m, 1h, 6h, 12h, 1d", interval)
	}

	return pgInterval, nil
}

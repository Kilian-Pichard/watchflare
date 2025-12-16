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

	// Use pre-calculated continuous aggregates for better performance
	tableName, err := getContinuousAggregateTable(params.Interval)
	if err != nil {
		return nil, err
	}

	// Query from the appropriate continuous aggregate view
	query := fmt.Sprintf(`
		SELECT
			bucket AS timestamp,
			cpu_usage_percent,
			CAST(memory_total_bytes AS BIGINT) AS memory_total_bytes,
			CAST(memory_used_bytes AS BIGINT) AS memory_used_bytes,
			CAST(memory_available_bytes AS BIGINT) AS memory_available_bytes,
			load_avg1_min AS load_avg_1min,
			load_avg5_min AS load_avg_5min,
			load_avg15_min AS load_avg_15min,
			CAST(disk_total_bytes AS BIGINT) AS disk_total_bytes,
			CAST(disk_used_bytes AS BIGINT) AS disk_used_bytes,
			CAST(uptime_seconds AS BIGINT) AS uptime_seconds
		FROM %s
		WHERE server_id = $1 AND bucket >= $2 AND bucket <= $3
		ORDER BY bucket ASC
	`, tableName)

	if err := database.DB.Raw(query, params.ServerID, params.Start, params.End).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// getContinuousAggregateTable returns the appropriate continuous aggregate table for the given interval
func getContinuousAggregateTable(interval string) (string, error) {
	// Map intervals to their continuous aggregate views
	aggregateTables := map[string]string{
		"10m": "metrics_10min", // Vue 12h
		"15m": "metrics_15min", // Vue 24h
		"2h":  "metrics_2h",    // Vue 7j
		"8h":  "metrics_8h",    // Vue 30j
	}

	tableName, ok := aggregateTables[interval]
	if !ok {
		return "", fmt.Errorf("invalid interval: %s. Valid intervals: 10m, 15m, 2h, 8h", interval)
	}

	return tableName, nil
}


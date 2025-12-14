package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Metric stores time-series system metrics for a server
type Metric struct {
	ID        string    `gorm:"type:char(36);primaryKey;priority:1" json:"id"`
	Timestamp time.Time `gorm:"primaryKey;priority:2;not null" json:"timestamp"` // Hypertable partition key
	ServerID  string    `gorm:"type:char(36);index:idx_metrics_server;not null" json:"server_id"`

	// CPU metrics
	CPUUsagePercent float64 `json:"cpu_usage_percent"`

	// Memory metrics (in bytes)
	MemoryTotalBytes     uint64 `json:"memory_total_bytes"`
	MemoryUsedBytes      uint64 `json:"memory_used_bytes"`
	MemoryAvailableBytes uint64 `json:"memory_available_bytes"`

	// Load average
	LoadAvg1Min  float64 `json:"load_avg_1min"`
	LoadAvg5Min  float64 `json:"load_avg_5min"`
	LoadAvg15Min float64 `json:"load_avg_15min"`

	// Disk metrics (in bytes)
	DiskTotalBytes uint64 `json:"disk_total_bytes"`
	DiskUsedBytes  uint64 `json:"disk_used_bytes"`

	// System uptime
	UptimeSeconds uint64 `json:"uptime_seconds"`

	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate hook to generate UUID before creating a metric
func (m *Metric) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for GORM
func (Metric) TableName() string {
	return "metrics"
}

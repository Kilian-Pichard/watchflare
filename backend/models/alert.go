package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Alert metric type constants.
const (
	MetricTypeServerDown  = "server_down"
	MetricTypeCPUUsage    = "cpu_usage"
	MetricTypeMemoryUsage = "memory_usage"
	MetricTypeDiskUsage   = "disk_usage"
	MetricTypeLoadAvg     = "load_avg"
	MetricTypeLoadAvg5    = "load_avg_5"
	MetricTypeLoadAvg15   = "load_avg_15"
	MetricTypeTemperature = "temperature"
)

// AllMetricTypes lists all valid metric types in display order.
var AllMetricTypes = []string{
	MetricTypeServerDown,
	MetricTypeCPUUsage,
	MetricTypeMemoryUsage,
	MetricTypeDiskUsage,
	MetricTypeLoadAvg,
	MetricTypeLoadAvg5,
	MetricTypeLoadAvg15,
	MetricTypeTemperature,
}

// AlertRule holds the global default threshold for a metric type.
type AlertRule struct {
	MetricType      string    `gorm:"primaryKey;type:varchar(20)" json:"metric_type"`
	Enabled         bool      `gorm:"not null;default:false" json:"enabled"`
	Threshold       float64   `gorm:"not null;default:0" json:"threshold"`
	DurationMinutes int       `gorm:"not null;default:5" json:"duration_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ServerAlertRule is a per-server override of a global alert rule.
type ServerAlertRule struct {
	ServerID        string    `gorm:"type:char(36);primaryKey" json:"server_id"`
	MetricType      string    `gorm:"type:varchar(20);primaryKey" json:"metric_type"`
	Enabled         bool      `gorm:"not null;default:false" json:"enabled"`
	Threshold       float64   `gorm:"not null;default:0" json:"threshold"`
	DurationMinutes int       `gorm:"not null;default:5" json:"duration_minutes"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AlertIncident tracks an active or resolved alert for a server.
type AlertIncident struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	ServerID       string     `gorm:"type:char(36);not null;index:idx_alert_incidents_server" json:"server_id"`
	MetricType     string     `gorm:"type:varchar(20);not null" json:"metric_type"`
	StartedAt      time.Time  `gorm:"not null" json:"started_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	Notified       bool       `gorm:"not null;default:false" json:"-"`
	ThresholdValue float64    `gorm:"not null;default:0" json:"threshold_value"`
	CurrentValue   float64    `gorm:"not null;default:0" json:"current_value"`
}

func (i *AlertIncident) BeforeCreate(_ *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

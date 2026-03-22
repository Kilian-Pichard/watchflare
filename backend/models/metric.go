package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SensorReading represents a single temperature sensor reading
type SensorReading struct {
	Key                string  `json:"key"`
	TemperatureCelsius float64 `json:"temperature_celsius"`
}

// SensorReadings is a slice of SensorReading with JSONB support for GORM
type SensorReadings []SensorReading

func (s SensorReadings) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *SensorReadings) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into SensorReadings", value)
	}
	return json.Unmarshal(bytes, s)
}

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

	// Disk I/O metrics (bytes per second)
	DiskReadBytesPerSec  uint64 `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec uint64 `json:"disk_write_bytes_per_sec"`

	// Network metrics (bytes per second)
	NetworkRxBytesPerSec uint64 `json:"network_rx_bytes_per_sec"`
	NetworkTxBytesPerSec uint64 `json:"network_tx_bytes_per_sec"`

	// Temperature (physical servers only, 0 if unavailable)
	CPUTemperatureCelsius float64 `json:"cpu_temperature_celsius"`

	// All sensor readings (temperature sensors, battery, storage, etc.)
	SensorReadings SensorReadings `gorm:"type:jsonb" json:"sensor_readings,omitempty"`

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

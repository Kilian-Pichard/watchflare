package models

import "time"

// SensorMetric represents a single sensor reading stored in the normalized hypertable.
type SensorMetric struct {
	Time        time.Time `json:"time"`
	ServerID    string    `json:"server_id"`
	SensorKey   string    `json:"sensor_key"`
	Temperature float64   `json:"temperature"`
}

// TableName overrides the GORM table name.
func (SensorMetric) TableName() string {
	return "sensor_metrics"
}

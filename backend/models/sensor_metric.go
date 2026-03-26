package models

import "time"

// SensorMetric represents a single sensor reading stored in the normalized hypertable.
type SensorMetric struct {
	Time        time.Time `gorm:"primaryKey;priority:1;not null" json:"time"`
	ServerID    string    `gorm:"type:char(36);primaryKey;priority:2;not null" json:"server_id"`
	SensorKey   string    `gorm:"primaryKey;priority:3;not null" json:"sensor_key"`
	Temperature float64   `gorm:"not null" json:"temperature"`
}


package models

import "time"

// Package represents current package state on a server
type Package struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	ServerID       string     `gorm:"type:char(36);not null" json:"server_id"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Version        string     `gorm:"type:varchar(100);not null" json:"version"`
	Architecture   string     `gorm:"type:varchar(50)" json:"architecture"`
	PackageManager string     `gorm:"type:varchar(20);not null" json:"package_manager"`
	Source         string     `gorm:"type:varchar(255)" json:"source"`
	InstalledAt    *time.Time `json:"installed_at"`
	PackageSize    int64      `json:"package_size"`
	Description    string     `gorm:"type:varchar(100)" json:"description"`
	FirstSeen      time.Time  `gorm:"not null;default:now()" json:"first_seen"`
	LastSeen       time.Time  `gorm:"not null;default:now()" json:"last_seen"`
}

// PackageHistory stores temporal snapshots of packages (TimescaleDB hypertable)
type PackageHistory struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	Timestamp      time.Time `gorm:"primaryKey;not null" json:"timestamp"`
	ServerID       string    `gorm:"type:char(36);not null" json:"server_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Version        string    `gorm:"type:varchar(100);not null" json:"version"`
	Architecture   string    `gorm:"type:varchar(50)" json:"architecture"`
	PackageManager string    `gorm:"type:varchar(20);not null" json:"package_manager"`
	Source         string    `gorm:"type:varchar(255)" json:"source"`
	PackageSize    int64     `json:"package_size"`
	Description    string    `gorm:"type:varchar(100)" json:"description"`
	ChangeType     string    `gorm:"type:varchar(20);not null" json:"change_type"` // 'added', 'removed', 'updated', 'initial'
}

// PackageCollection tracks metadata about package collection jobs
type PackageCollection struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	ServerID       string    `gorm:"type:char(36);not null" json:"server_id"`
	Timestamp      time.Time `gorm:"not null;default:now()" json:"timestamp"`
	CollectionType string    `gorm:"type:varchar(20);not null" json:"collection_type"` // 'full', 'delta', 'initial'
	PackageCount   int       `gorm:"not null" json:"package_count"`
	ChangesCount   int       `gorm:"default:0" json:"changes_count"`
	DurationMs     int       `json:"duration_ms"`
	Status         string    `gorm:"type:varchar(20);not null;default:'success'" json:"status"` // 'success', 'failed', 'partial'
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
}

// TableName overrides for GORM
func (Package) TableName() string {
	return "packages"
}

func (PackageHistory) TableName() string {
	return "package_history"
}

func (PackageCollection) TableName() string {
	return "package_collections"
}

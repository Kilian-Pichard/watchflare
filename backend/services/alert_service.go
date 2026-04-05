package services

import (
	"errors"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AlertRuleInput carries data for creating or updating an alert rule.
type AlertRuleInput struct {
	MetricType      string
	Enabled         bool
	Threshold       float64
	DurationMinutes int
}

// EffectiveAlertRule is the merged view of a rule for a specific server.
type EffectiveAlertRule struct {
	MetricType      string  `json:"metric_type"`
	Enabled         bool    `json:"enabled"`
	Threshold       float64 `json:"threshold"`
	DurationMinutes int     `json:"duration_minutes"`
	IsOverride      bool    `json:"is_override"` // true when a server-level rule exists
}

// GetAlertRules returns all global alert rules in canonical order.
func GetAlertRules() ([]models.AlertRule, error) {
	var rules []models.AlertRule
	if err := database.DB.Find(&rules).Error; err != nil {
		return nil, err
	}
	// Return in AllMetricTypes order so the UI can rely on it.
	byType := make(map[string]models.AlertRule, len(rules))
	for _, r := range rules {
		byType[r.MetricType] = r
	}
	ordered := make([]models.AlertRule, 0, len(models.AllMetricTypes))
	for _, mt := range models.AllMetricTypes {
		if r, ok := byType[mt]; ok {
			ordered = append(ordered, r)
		}
	}
	return ordered, nil
}

// UpdateAlertRules saves a batch of global alert rule changes.
func UpdateAlertRules(inputs []AlertRuleInput) error {
	now := time.Now()
	for _, in := range inputs {
		rule := models.AlertRule{
			MetricType:      in.MetricType,
			Enabled:         in.Enabled,
			Threshold:       in.Threshold,
			DurationMinutes: in.DurationMinutes,
			UpdatedAt:       now,
		}
		if err := database.DB.Save(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetServerAlertRules returns the effective (merged) rules for a server,
// indicating which ones have a server-level override.
func GetServerAlertRules(serverID string) ([]EffectiveAlertRule, error) {
	globals, err := GetAlertRules()
	if err != nil {
		return nil, err
	}

	var overrides []models.ServerAlertRule
	if err := database.DB.Where("server_id = ?", serverID).Find(&overrides).Error; err != nil {
		return nil, err
	}
	overrideMap := make(map[string]models.ServerAlertRule, len(overrides))
	for _, o := range overrides {
		overrideMap[o.MetricType] = o
	}

	result := make([]EffectiveAlertRule, 0, len(globals))
	for _, g := range globals {
		if o, ok := overrideMap[g.MetricType]; ok {
			result = append(result, EffectiveAlertRule{
				MetricType:      g.MetricType,
				Enabled:         o.Enabled,
				Threshold:       o.Threshold,
				DurationMinutes: o.DurationMinutes,
				IsOverride:      true,
			})
		} else {
			result = append(result, EffectiveAlertRule{
				MetricType:      g.MetricType,
				Enabled:         g.Enabled,
				Threshold:       g.Threshold,
				DurationMinutes: g.DurationMinutes,
				IsOverride:      false,
			})
		}
	}
	return result, nil
}

// UpsertServerAlertRule creates or updates a per-server alert rule override.
func UpsertServerAlertRule(serverID, metricType string, input AlertRuleInput) error {
	rule := models.ServerAlertRule{
		ServerID:        serverID,
		MetricType:      metricType,
		Enabled:         input.Enabled,
		Threshold:       input.Threshold,
		DurationMinutes: input.DurationMinutes,
		UpdatedAt:       time.Now(),
	}
	return database.DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&rule).Error
}

// DeleteServerAlertRule removes a per-server override, reverting to the global default.
func DeleteServerAlertRule(serverID, metricType string) error {
	err := database.DB.
		Where("server_id = ? AND metric_type = ?", serverID, metricType).
		Delete(&models.ServerAlertRule{}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

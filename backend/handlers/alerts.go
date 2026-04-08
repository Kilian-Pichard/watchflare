package handlers

import (
	"log/slog"
	"net/http"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// GetAlertRules returns the global alert rules.
func GetAlertRules(c *gin.Context) {
	rules, err := services.GetAlertRules()
	if err != nil {
		slog.Error("failed to get alert rules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get alert rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// UpdateAlertRulesRequest is the body for PUT /settings/alerts.
type UpdateAlertRulesRequest struct {
	Rules []UpdateAlertRuleItem `json:"rules"`
}

// UpdateAlertRuleItem is one entry in an UpdateAlertRulesRequest.
type UpdateAlertRuleItem struct {
	MetricType      string  `json:"metric_type"`
	Enabled         bool    `json:"enabled"`
	Threshold       float64 `json:"threshold"`
	DurationMinutes int     `json:"duration_minutes"`
}

// UpdateAlertRules replaces all global alert rules.
func UpdateAlertRules(c *gin.Context) {
	var req UpdateAlertRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range req.Rules {
		if !isValidMetricType(item.MetricType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric_type: " + item.MetricType})
			return
		}
		if item.DurationMinutes < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duration_minutes must be at least 1"})
			return
		}
	}

	inputs := make([]services.AlertRuleInput, len(req.Rules))
	for i, r := range req.Rules {
		inputs[i] = services.AlertRuleInput{
			MetricType:      r.MetricType,
			Enabled:         r.Enabled,
			Threshold:       r.Threshold,
			DurationMinutes: r.DurationMinutes,
		}
	}

	if err := services.UpdateAlertRules(inputs); err != nil {
		slog.Error("failed to update alert rules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alert rules updated"})
}

// GetServerAlertRules returns the effective alert rules for a specific server.
func GetServerAlertRules(c *gin.Context) {
	serverID := c.Param("id")
	rules, err := services.GetServerAlertRules(serverID)
	if err != nil {
		slog.Error("failed to get server alert rules", "server_id", serverID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get server alert rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// UpsertServerAlertRuleRequest is the body for PUT /servers/:id/alerts/:metric_type.
type UpsertServerAlertRuleRequest struct {
	Enabled         bool    `json:"enabled"`
	Threshold       float64 `json:"threshold"`
	DurationMinutes int     `json:"duration_minutes"`
}

// UpsertServerAlertRule creates or updates a per-server alert rule override.
func UpsertServerAlertRule(c *gin.Context) {
	serverID := c.Param("id")
	metricType := c.Param("metric_type")

	if !isValidMetricType(metricType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric_type: " + metricType})
		return
	}

	var req UpsertServerAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DurationMinutes < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duration_minutes must be at least 1"})
		return
	}

	if err := services.UpsertServerAlertRule(serverID, metricType, services.AlertRuleInput{
		MetricType:      metricType,
		Enabled:         req.Enabled,
		Threshold:       req.Threshold,
		DurationMinutes: req.DurationMinutes,
	}); err != nil {
		slog.Error("failed to upsert server alert rule", "server_id", serverID, "metric_type", metricType, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save server alert rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "server alert rule saved"})
}

// DeleteServerAlertRule removes a per-server override, reverting to the global default.
func DeleteServerAlertRule(c *gin.Context) {
	serverID := c.Param("id")
	metricType := c.Param("metric_type")

	if !isValidMetricType(metricType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric_type: " + metricType})
		return
	}

	if err := services.DeleteServerAlertRule(serverID, metricType); err != nil {
		slog.Error("failed to delete server alert rule", "server_id", serverID, "metric_type", metricType, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete server alert rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "server alert rule deleted"})
}

// ActiveIncidentItem is the response shape for GET /alerts/active.
type ActiveIncidentItem struct {
	ID             string    `json:"id"`
	ServerID       string    `json:"server_id"`
	ServerName     string    `json:"server_name"`
	MetricType     string    `json:"metric_type"`
	StartedAt      time.Time `json:"started_at"`
	ThresholdValue float64   `json:"threshold_value"`
	CurrentValue   float64   `json:"current_value"`
}

// GetActiveIncidents returns all unresolved alert incidents with their server name.
func GetActiveIncidents(c *gin.Context) {
	var items []ActiveIncidentItem
	err := database.DB.Table("alert_incidents").
		Select("alert_incidents.id, alert_incidents.server_id, servers.name AS server_name, alert_incidents.metric_type, alert_incidents.started_at, alert_incidents.threshold_value, alert_incidents.current_value").
		Joins("JOIN servers ON servers.id = alert_incidents.server_id").
		Where("alert_incidents.resolved_at IS NULL").
		Order("alert_incidents.started_at DESC").
		Scan(&items).Error
	if err != nil {
		slog.Error("failed to get active incidents", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active incidents"})
		return
	}
	if items == nil {
		items = []ActiveIncidentItem{}
	}
	c.JSON(http.StatusOK, gin.H{"incidents": items})
}

func isValidMetricType(mt string) bool {
	for _, valid := range models.AllMetricTypes {
		if mt == valid {
			return true
		}
	}
	return false
}

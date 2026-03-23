package handlers

import (
	"net/http"
	"time"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// GetMetrics returns metrics for a specific server
func GetMetrics(c *gin.Context) {
	serverID := c.Param("id")

	// Parse query parameters
	startStr := c.DefaultQuery("start", "")
	endStr := c.DefaultQuery("end", "")
	interval := c.DefaultQuery("interval", "")
	timeRange := c.DefaultQuery("time_range", "")

	// Default to last hour if no time range specified
	var start, end time.Time
	var err error

	// If time_range is provided, calculate start/end and map to interval
	if timeRange != "" {
		end = time.Now()

		// Map time_range to duration and interval
		switch timeRange {
		case "1h":
			start = end.Add(-1 * time.Hour)
			interval = "" // Raw data (every 30s) - 120 points
		case "12h":
			start = end.Add(-12 * time.Hour)
			interval = "10m" // Continuous aggregate 10min - 72 points
		case "24h":
			start = end.Add(-24 * time.Hour)
			interval = "15m" // Continuous aggregate 15min - 96 points
		case "7d":
			start = end.Add(-7 * 24 * time.Hour)
			interval = "2h" // Continuous aggregate 2h - 84 points
		case "30d":
			start = end.Add(-30 * 24 * time.Hour)
			interval = "8h" // Continuous aggregate 8h - 90 points
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time_range. Valid values: 1h, 12h, 24h, 7d, 30d"})
			return
		}
	} else {
		// Legacy behavior: use start/end directly
		if endStr == "" {
			end = time.Now()
		} else {
			end, err = parseTime(endStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format. Use RFC3339 or Unix timestamp"})
				return
			}
		}

		if startStr == "" {
			start = end.Add(-1 * time.Hour) // Default: last hour
		} else {
			start, err = parseTime(startStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format. Use RFC3339 or Unix timestamp"})
				return
			}
		}
	}

	// Validate time range
	if start.After(end) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start time must be before end time"})
		return
	}

	// Query metrics
	params := services.MetricsQueryParams{
		ServerID: serverID,
		Start:    start,
		End:      end,
		Interval: interval,
	}

	metrics, err := services.GetMetrics(params)
	if err != nil {
		if err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":  serverID,
		"start":      start.Format(time.RFC3339),
		"end":        end.Format(time.RFC3339),
		"time_range": timeRange,
		"interval":   interval,
		"count":      len(metrics),
		"metrics":    metrics,
	})
}

// GetContainerMetrics returns container metrics for a specific server
func GetContainerMetrics(c *gin.Context) {
	serverID := c.Param("id")
	timeRange := c.DefaultQuery("time_range", "1h")

	end := time.Now()
	var start time.Time
	var interval string

	switch timeRange {
	case "1h":
		start = end.Add(-1 * time.Hour)
		interval = "" // Raw data
	case "12h":
		start = end.Add(-12 * time.Hour)
		interval = "10m"
	case "24h":
		start = end.Add(-24 * time.Hour)
		interval = "15m"
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
		interval = "2h"
	case "30d":
		start = end.Add(-30 * 24 * time.Hour)
		interval = "8h"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time_range. Valid values: 1h, 12h, 24h, 7d, 30d"})
		return
	}

	metrics, err := services.GetContainerMetrics(serverID, start, end, interval)
	if err != nil {
		if err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":  serverID,
		"time_range": timeRange,
		"count":      len(metrics),
		"metrics":    metrics,
	})
}

// GetSensorReadings returns per-sensor temperature data for a specific server
func GetSensorReadings(c *gin.Context) {
	serverID := c.Param("id")
	timeRange := c.DefaultQuery("time_range", "1h")

	end := time.Now()
	var start time.Time
	var interval string

	switch timeRange {
	case "1h":
		start = end.Add(-1 * time.Hour)
		interval = ""
	case "12h":
		start = end.Add(-12 * time.Hour)
		interval = "10m"
	case "24h":
		start = end.Add(-24 * time.Hour)
		interval = "15m"
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
		interval = "2h"
	case "30d":
		start = end.Add(-30 * 24 * time.Hour)
		interval = "8h"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_range. valid values: 1h, 12h, 24h, 7d, 30d"})
		return
	}

	data, err := services.GetSensorReadings(serverID, start, end, interval)
	if err != nil {
		if err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id":  serverID,
		"time_range": timeRange,
		"count":      len(data),
		"data":       data,
	})
}

// parseTime attempts to parse time from RFC3339 format or Unix timestamp
func parseTime(timeStr string) (time.Time, error) {
	// Try RFC3339 format first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try Unix timestamp
	var timestamp int64
	if _, err := time.Parse("2006-01-02T15:04:05Z07:00", timeStr); err != nil {
		// If not a valid timestamp format, try parsing as Unix timestamp
		if n, err := time.Parse("1136239445", timeStr); err == nil {
			return n, nil
		}
	}

	// Try parsing as integer Unix timestamp
	if _, err := time.ParseDuration(timeStr); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, err
}

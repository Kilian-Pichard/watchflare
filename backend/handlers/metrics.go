package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// resolveTimeRange maps a time_range string to start time, end time, and query interval.
// Returns ok=false for unknown values.
func resolveTimeRange(timeRange string) (start, end time.Time, interval string, ok bool) {
	end = time.Now()
	switch timeRange {
	case "1h":
		start = end.Add(-time.Hour)
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
		return
	}
	ok = true
	return
}

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
		var ok bool
		start, end, interval, ok = resolveTimeRange(timeRange)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_range, valid values: 1h, 12h, 24h, 7d, 30d"})
			return
		}
	} else {
		// Legacy behavior: use start/end directly
		if endStr == "" {
			end = time.Now()
		} else {
			end, err = parseTime(endStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, use RFC3339 or unix timestamp"})
				return
			}
		}

		if startStr == "" {
			start = end.Add(-1 * time.Hour) // Default: last hour
		} else {
			start, err = parseTime(startStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, use RFC3339 or unix timestamp"})
				return
			}
		}
	}

	// Validate time range
	if start.After(end) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start time must be before end time"})
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
		if errors.Is(err, services.ErrServerNotFound) {
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

	start, end, interval, ok := resolveTimeRange(timeRange)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_range, valid values: 1h, 12h, 24h, 7d, 30d"})
		return
	}

	metrics, err := services.GetContainerMetrics(serverID, start, end, interval)
	if err != nil {
		if errors.Is(err, services.ErrServerNotFound) {
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

	start, end, interval, ok := resolveTimeRange(timeRange)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_range, valid values: 1h, 12h, 24h, 7d, 30d"})
		return
	}

	data, err := services.GetSensorReadings(serverID, start, end, interval)
	if err != nil {
		if errors.Is(err, services.ErrServerNotFound) {
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

// parseTime attempts to parse time from RFC3339 format or Unix timestamp (seconds).
func parseTime(timeStr string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}
	if ts, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}

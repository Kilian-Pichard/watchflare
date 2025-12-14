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

	// Default to last hour if no time range specified
	var start, end time.Time
	var err error

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
		"server_id": serverID,
		"start":     start.Format(time.RFC3339),
		"end":       end.Format(time.RFC3339),
		"interval":  interval,
		"count":     len(metrics),
		"metrics":   metrics,
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

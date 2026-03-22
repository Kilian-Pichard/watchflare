package services

import (
	"log/slog"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/sse"
)

// AggregatedMetricsScheduler handles periodic calculation and broadcasting of aggregated metrics
type AggregatedMetricsScheduler struct {
	interval time.Duration
	ticker   *time.Ticker
	stopChan chan bool
}

// NewAggregatedMetricsScheduler creates a new scheduler with the given interval
func NewAggregatedMetricsScheduler(interval time.Duration) *AggregatedMetricsScheduler {
	return &AggregatedMetricsScheduler{
		interval: interval,
		stopChan: make(chan bool),
	}
}

// Start begins the scheduler - calculates and broadcasts aggregated metrics at regular intervals
func (s *AggregatedMetricsScheduler) Start() {
	// Wait 2 seconds after each bucket boundary to ensure agents have sent their metrics
	// This gives agents time to send (typically 50-500ms) while keeping latency minimal
	const processingDelay = 2 * time.Second

	slog.Info("aggregated metrics scheduler starting", "interval", s.interval, "processing_delay", processingDelay)

	// Run in a loop, recalculating alignment on each iteration
	for {
		// Calculate delay to next interval boundary + processing delay
		now := time.Now()
		nextBoundary := now.Truncate(s.interval).Add(s.interval)
		nextTick := nextBoundary.Add(processingDelay)
		delay := nextTick.Sub(now)

		// Wait until the next aligned boundary + processing delay
		select {
		case <-time.After(delay):
			// Calculate and broadcast
			s.calculateAndBroadcast()
		case <-s.stopChan:
			slog.Info("aggregated metrics scheduler stopped")
			return
		}
	}
}

// Stop stops the scheduler
func (s *AggregatedMetricsScheduler) Stop() {
	s.stopChan <- true
}

// calculateAndBroadcast calculates aggregated metrics for the last interval and broadcasts via SSE
func (s *AggregatedMetricsScheduler) calculateAndBroadcast() {
	// Calculate the bucket time for the interval that just completed
	// Example: if now is 18:34:05, we calculate metrics for 18:34:00 bucket
	// The bucket window is (18:33:30, 18:34:00]
	now := time.Now()
	bucketTime := now.Truncate(s.interval)

	// Calculate time window for the bucket
	// For 30s interval: if bucket is 18:34:00, window is (18:33:30, 18:34:00]
	endTime := bucketTime
	startTime := bucketTime.Add(-s.interval)

	// Query aggregated metrics for the interval
	query := `
		SELECT
			COALESCE(AVG(m.cpu_usage_percent), 0) as cpu_usage_percent,
			COALESCE(SUM(m.memory_total_bytes), 0) as memory_total_bytes,
			COALESCE(SUM(m.memory_used_bytes), 0) as memory_used_bytes,
			COALESCE(SUM(CASE WHEN s.environment_type != 'container' THEN m.disk_total_bytes ELSE 0 END), 0) as disk_total_bytes,
			COALESCE(SUM(CASE WHEN s.environment_type != 'container' THEN m.disk_used_bytes ELSE 0 END), 0) as disk_used_bytes
		FROM metrics m
		JOIN servers s ON m.server_id = s.id
		WHERE s.status = 'online'
		  AND m.timestamp > $1
		  AND m.timestamp <= $2
	`

	var cpuUsagePercent float64
	var memoryTotalBytes uint64
	var memoryUsedBytes uint64
	var diskTotalBytes uint64
	var diskUsedBytes uint64

	err := database.DB.Raw(query, startTime, endTime).Row().Scan(
		&cpuUsagePercent,
		&memoryTotalBytes,
		&memoryUsedBytes,
		&diskTotalBytes,
		&diskUsedBytes,
	)

	if err != nil {
		slog.Error("failed to calculate aggregated metrics", "error", err)
		return
	}

	// Skip broadcasting if no data (all servers paused/offline)
	if memoryTotalBytes == 0 && diskTotalBytes == 0 && cpuUsagePercent == 0 {
		return
	}

	// Create aggregated metrics update
	update := sse.AggregatedMetricsUpdate{
		Timestamp:            bucketTime.Format(time.RFC3339),
		CPUUsagePercent:      cpuUsagePercent,
		MemoryTotalBytes:     memoryTotalBytes,
		MemoryUsedBytes:      memoryUsedBytes,
		MemoryAvailableBytes: memoryTotalBytes - memoryUsedBytes,
		DiskTotalBytes:       diskTotalBytes,
		DiskUsedBytes:        diskUsedBytes,
	}

	// Broadcast via SSE
	broker := sse.GetBroker()
	broker.BroadcastAggregatedMetricsUpdate(update)
}

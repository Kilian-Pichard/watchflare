package services

import (
	"log"
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
	// Calculate initial delay to align with interval boundaries
	// For 30s interval: if current time is 07:10:23, wait 7s to start at 07:10:30
	now := time.Now()
	nextTick := now.Truncate(s.interval).Add(s.interval)
	initialDelay := nextTick.Sub(now)

	log.Printf("Aggregated metrics scheduler starting in %v (next tick at %s)", initialDelay, nextTick.Format("15:04:05"))

	// Wait for initial alignment
	time.Sleep(initialDelay)

	// Create ticker that fires at aligned intervals
	s.ticker = time.NewTicker(s.interval)

	// Calculate and broadcast immediately on first aligned tick
	s.calculateAndBroadcast()

	// Then continue with ticker
	for {
		select {
		case <-s.ticker.C:
			s.calculateAndBroadcast()
		case <-s.stopChan:
			s.ticker.Stop()
			log.Println("Aggregated metrics scheduler stopped")
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
	// Calculate time window: (now - interval, now]
	// Using interval exclusion at start: timestamp > startTime AND timestamp <= endTime
	endTime := time.Now()
	startTime := endTime.Add(-s.interval)

	// Round endTime to bucket boundary (e.g., 07:10:30)
	bucketTime := endTime.Truncate(s.interval)

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
		log.Printf("Error calculating aggregated metrics: %v", err)
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

	log.Printf("Broadcasted aggregated metrics for bucket %s (CPU: %.2f%%, Memory: %d/%d, Disk: %d/%d)",
		bucketTime.Format("15:04:05"),
		cpuUsagePercent,
		memoryUsedBytes,
		memoryTotalBytes,
		diskUsedBytes,
		diskTotalBytes,
	)
}

package services

import (
	"context"
	"log/slog"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/sse"
)

// AggregatedMetricsScheduler periodically calculates and broadcasts aggregated metrics.
type AggregatedMetricsScheduler struct {
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewAggregatedMetricsScheduler(interval time.Duration) *AggregatedMetricsScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &AggregatedMetricsScheduler{
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start runs the scheduler. Call in a goroutine.
// It waits for the next interval boundary plus a 2s processing delay before each run,
// giving agents time to deliver their metrics (typically 50–500ms).
func (s *AggregatedMetricsScheduler) Start() {
	const processingDelay = 2 * time.Second

	slog.Info("aggregated metrics scheduler starting", "interval", s.interval, "processing_delay", processingDelay)

	for {
		now := time.Now()
		nextTick := now.Truncate(s.interval).Add(s.interval).Add(processingDelay)
		delay := nextTick.Sub(now)

		select {
		case <-time.After(delay):
			s.calculateAndBroadcast()
		case <-s.ctx.Done():
			slog.Info("aggregated metrics scheduler stopped")
			return
		}
	}
}

func (s *AggregatedMetricsScheduler) Stop() {
	s.cancel()
}

func (s *AggregatedMetricsScheduler) calculateAndBroadcast() {
	// Compute the bucket that just completed.
	// Example: if now is 18:34:05 and interval is 30s,
	// bucket = 18:34:00, window = (18:33:30, 18:34:00].
	now := time.Now()
	bucketTime := now.Truncate(s.interval)
	endTime := bucketTime
	startTime := bucketTime.Add(-s.interval)

	query := `
		SELECT
			COALESCE(AVG(m.cpu_usage_percent), 0) as cpu_usage_percent,
			COALESCE(SUM(m.memory_total_bytes), 0) as memory_total_bytes,
			COALESCE(SUM(m.memory_used_bytes), 0) as memory_used_bytes,
			COALESCE(SUM(CASE WHEN s.environment_type != 'container' THEN m.disk_total_bytes ELSE 0 END), 0) as disk_total_bytes,
			COALESCE(SUM(CASE WHEN s.environment_type != 'container' THEN m.disk_used_bytes ELSE 0 END), 0) as disk_used_bytes
		FROM metrics m
		JOIN hosts s ON m.host_id = s.id
		WHERE s.status = 'online'
		  AND m.timestamp > $1
		  AND m.timestamp <= $2
	`

	var cpuUsagePercent float64
	var memoryTotalBytes, memoryUsedBytes, diskTotalBytes, diskUsedBytes uint64

	if err := database.DB.Raw(query, startTime, endTime).Row().Scan(
		&cpuUsagePercent, &memoryTotalBytes, &memoryUsedBytes, &diskTotalBytes, &diskUsedBytes,
	); err != nil {
		slog.Error("failed to calculate aggregated metrics", "error", err)
		return
	}

	// Skip broadcasting when all hosts are paused or offline.
	if memoryTotalBytes == 0 && diskTotalBytes == 0 && cpuUsagePercent == 0 {
		return
	}

	sse.GetBroker().BroadcastAggregatedMetricsUpdate(sse.AggregatedMetricsUpdate{
		Timestamp:            bucketTime.Format(time.RFC3339),
		CPUUsagePercent:      cpuUsagePercent,
		MemoryTotalBytes:     memoryTotalBytes,
		MemoryUsedBytes:      memoryUsedBytes,
		MemoryAvailableBytes: memoryTotalBytes - memoryUsedBytes,
		DiskTotalBytes:       diskTotalBytes,
		DiskUsedBytes:        diskUsedBytes,
	})
}

package wal

import (
	"context"
	"fmt"
	"log"
	"time"

	"watchflare-agent/client"
	"watchflare-agent/errors"
	"watchflare-agent/metrics"
	"watchflare-agent/sysinfo"
	pb "watchflare/shared/proto"

	"google.golang.org/protobuf/proto"
)

// Sender manages metrics collection and sending with WAL persistence
type Sender struct {
	wal             *WAL
	client          *client.Client
	agentID         string
	agentKey        string
	metricsInterval time.Duration
	maxWALSize      int64
	metricsConfig   *sysinfo.MetricsConfig
}

// NewSender creates a new Sender
func NewSender(wal *WAL, grpcClient *client.Client, agentID, agentKey string, metricsIntervalSec int, maxWALSizeMB int, metricsConfig *sysinfo.MetricsConfig) *Sender {
	return &Sender{
		wal:             wal,
		client:          grpcClient,
		agentID:         agentID,
		agentKey:        agentKey,
		metricsInterval: time.Duration(metricsIntervalSec) * time.Second,
		maxWALSize:      int64(maxWALSizeMB) * 1024 * 1024,
		metricsConfig:   metricsConfig,
	}
}

// Run starts the sender loop (blocks until context cancelled)
func (s *Sender) Run(ctx context.Context) error {
	log.Println("Sender starting...")

	// Check WAL size and truncate if needed BEFORE replaying
	if s.wal != nil {
		size, err := s.wal.Size()
		if err != nil {
			log.Printf("Warning: Failed to check WAL size on startup: %v", err)
		} else if size > s.maxWALSize {
			log.Printf("WAL size %.2f MB exceeds max %d MB on startup, truncating (FIFO)...",
				float64(size)/1024/1024, s.maxWALSize/1024/1024)
			if err := s.wal.Truncate(); err != nil {
				log.Printf("Warning: Failed to truncate WAL on startup: %v", err)
			} else {
				newSize, _ := s.wal.Size()
				log.Printf("✓ WAL truncated on startup: %.2f MB → %.2f MB",
					float64(size)/1024/1024, float64(newSize)/1024/1024)
			}
		}
	}

	// Replay WAL on startup (after potential truncation)
	if err := s.replayWAL(); err != nil {
		log.Printf("Warning: Failed to replay WAL: %v", err)
	}

	// Align first tick to the next wall clock boundary (e.g., :00, :30 for 30s interval)
	// This ensures all agents produce timestamps at consistent round intervals
	now := time.Now()
	intervalSec := int64(s.metricsInterval.Seconds())
	nextBoundary := time.Unix(((now.Unix()/intervalSec)+1)*intervalSec, 0)
	waitDuration := time.Until(nextBoundary)

	log.Printf("Sender aligning to clock boundary (waiting %v until %s)", waitDuration, nextBoundary.Format("15:04:05"))

	select {
	case <-time.After(waitDuration):
		// Aligned — start ticker BEFORE first collection so it stays on boundary
	case <-ctx.Done():
		log.Println("Sender shutting down before first collection")
		return nil
	}

	ticker := time.NewTicker(s.metricsInterval)
	defer ticker.Stop()

	log.Printf("Sender started (interval: %v)", s.metricsInterval)

	// First collection at boundary (ticker will fire next at boundary + interval)
	s.collectAndSend()

	for {
		select {
		case <-ticker.C:
			s.collectAndSend()

		case <-ctx.Done():
			log.Println("Sender shutting down...")
			s.shutdown()
			return nil
		}
	}
}

// replayWAL sends any pending metrics from WAL on startup
func (s *Sender) replayWAL() error {
	// Skip if WAL is disabled
	if s.wal == nil {
		return nil
	}

	records, err := s.wal.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read WAL: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	log.Printf("WAL RECOVERY: Found %d pending metrics from previous backend downtime", len(records))
	log.Printf("Sending accumulated metrics to backend...")

	// Try to send all pending records (no container metrics during WAL replay)
	success := true
	for i, data := range records {
		if err := s.sendRecord(data, false, nil); err != nil {
			if errors.IsTimestampError(err) {
				log.Printf("Failed to send record %d/%d: CLOCK SYNC ERROR - System time is out of sync (>5min difference with backend). "+
					"Fix: Run 'sudo timedatectl set-ntp true' and restart the agent (will retry later)", i+1, len(records))
			} else {
				log.Printf("Failed to send record %d/%d: %v (will retry later)", i+1, len(records), err)
			}
			success = false
			break // Stop at first failure
		}
	}

	// Clear WAL only if all sends succeeded
	if success {
		if err := s.wal.Clear(); err != nil {
			return fmt.Errorf("failed to clear WAL: %w", err)
		}
		log.Printf("✓ WAL RECOVERY COMPLETE: All %d pending metrics sent successfully", len(records))
		log.Printf("✓ WAL cleared (recovery finished)")
	}

	return nil
}

// collectAndSend collects metrics, persists to WAL, and sends
func (s *Sender) collectAndSend() {
	// 1. Collect metrics (using environment-based config)
	m, err := metrics.Collect(s.metricsConfig)
	if err != nil {
		log.Printf("Failed to collect metrics: %v", err)
		return
	}

	// Round timestamp to nearest interval boundary (e.g., 10:00:34 → 10:00:30)
	// This absorbs collection time (~1s for CPU) and produces clean aligned timestamps
	intervalSec := int64(s.metricsInterval.Seconds())
	m.Timestamp = ((m.Timestamp + intervalSec/2) / intervalSec) * intervalSec

	// Container metrics are sent with the current metrics only (not WAL'd)
	// They are point-in-time snapshots; stale container data is useless
	containerMetrics := m.ContainerMetrics
	m.ContainerMetrics = nil // Don't include in WAL serialization

	// If WAL is disabled, send directly
	if s.wal == nil {
		// Attach container metrics for direct send
		m.ContainerMetrics = containerMetrics
		if err := s.client.SendMetrics(s.agentID, s.agentKey, m); err != nil {
			log.Printf("Send failed: %v (metrics lost - WAL disabled)", err)
		} else {
			log.Printf("✓ Metrics sent (CPU: %.1f%%, Mem: %d/%d MB)",
				m.CPUUsagePercent,
				m.MemoryUsedBytes/1024/1024,
				m.MemoryTotalBytes/1024/1024)
		}
		return
	}

	// 2. Serialize to protobuf
	data, err := s.serializeMetrics(m)
	if err != nil {
		log.Printf("Failed to serialize metrics: %v", err)
		return
	}

	// 3. Append to WAL
	if err := s.wal.Append(data); err != nil {
		log.Printf("Failed to append to WAL: %v", err)
		return
	}

	// 4. Check WAL size and truncate if needed (FIFO)
	size, err := s.wal.Size()
	if err != nil {
		log.Printf("Warning: Failed to check WAL size: %v", err)
	} else if size > s.maxWALSize {
		log.Printf("WAL size %d MB exceeds max %d MB, truncating (FIFO)...",
			size/1024/1024, s.maxWALSize/1024/1024)
		if err := s.wal.Truncate(); err != nil {
			log.Printf("Warning: Failed to truncate WAL: %v", err)
		} else {
			newSize, _ := s.wal.Size()
			log.Printf("✓ WAL truncated: %d MB → %d MB", size/1024/1024, newSize/1024/1024)
		}
	}

	// 5. Try to send ALL pending metrics (including the one we just appended)
	records, err := s.wal.ReadAll()
	if err != nil {
		log.Printf("Failed to read WAL: %v", err)
		return
	}

	// Send all records; attach container metrics to the LAST record only (current metrics)
	success := true
	for i, record := range records {
		isLastRecord := i == len(records)-1
		if err := s.sendRecord(record, isLastRecord, containerMetrics); err != nil {
			if errors.IsTimestampError(err) {
				log.Printf("Send failed (record %d/%d): CLOCK SYNC ERROR - System time is out of sync (>5min difference with backend). "+
					"Fix: Run 'sudo timedatectl set-ntp true' and restart the agent (will retry in %v)",
					i+1, len(records), s.metricsInterval)
			} else {
				log.Printf("Send failed (record %d/%d): %v (will retry in %v)",
					i+1, len(records), err, s.metricsInterval)
			}
			success = false
			break // Stop at first failure
		}
	}

	// Clear WAL only if all sends succeeded
	if success {
		if err := s.wal.Clear(); err != nil {
			log.Printf("Warning: Failed to clear WAL: %v", err)
		} else {
			if len(records) > 1 {
				log.Printf("✓ Sent %d metrics (including %d accumulated during backend outage)",
					len(records), len(records)-1)
				log.Printf("✓ WAL cleared")
			}
			log.Printf("✓ Metrics sent (CPU: %.1f%%, Mem: %d/%d MB)",
				m.CPUUsagePercent,
				m.MemoryUsedBytes/1024/1024,
				m.MemoryTotalBytes/1024/1024)
		}
	}
}

// sendRecord sends a single serialized metrics record
// If includeContainers is true and containerMetrics is non-nil, they are attached to the send
func (s *Sender) sendRecord(data []byte, includeContainers bool, containerMetrics []metrics.ContainerMetric) error {
	// Deserialize
	var pbMetrics pb.Metrics
	if err := proto.Unmarshal(data, &pbMetrics); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Convert to metrics.SystemMetrics for client
	m := &metrics.SystemMetrics{
		CPUUsagePercent:       pbMetrics.CpuUsagePercent,
		MemoryTotalBytes:      pbMetrics.MemoryTotalBytes,
		MemoryUsedBytes:       pbMetrics.MemoryUsedBytes,
		MemoryAvailableBytes:  pbMetrics.MemoryAvailableBytes,
		LoadAvg1Min:           pbMetrics.LoadAvg_1Min,
		LoadAvg5Min:           pbMetrics.LoadAvg_5Min,
		LoadAvg15Min:          pbMetrics.LoadAvg_15Min,
		DiskTotalBytes:        pbMetrics.DiskTotalBytes,
		DiskUsedBytes:         pbMetrics.DiskUsedBytes,
		DiskReadBytesPerSec:   pbMetrics.DiskReadBytesPerSec,
		DiskWriteBytesPerSec:  pbMetrics.DiskWriteBytesPerSec,
		NetworkRxBytesPerSec:  pbMetrics.NetworkRxBytesPerSec,
		NetworkTxBytesPerSec:  pbMetrics.NetworkTxBytesPerSec,
		CPUTemperatureCelsius: pbMetrics.CpuTemperatureCelsius,
		UptimeSeconds:         pbMetrics.UptimeSeconds,
		Timestamp:             pbMetrics.Timestamp,
	}

	// Attach container metrics only to the most recent (current) record
	if includeContainers && len(containerMetrics) > 0 {
		m.ContainerMetrics = containerMetrics
	}

	// Send via gRPC
	return s.client.SendMetrics(s.agentID, s.agentKey, m)
}

// serializeMetrics converts SystemMetrics to protobuf bytes
func (s *Sender) serializeMetrics(m *metrics.SystemMetrics) ([]byte, error) {
	pbMetrics := &pb.Metrics{
		CpuUsagePercent:       m.CPUUsagePercent,
		MemoryTotalBytes:      m.MemoryTotalBytes,
		MemoryUsedBytes:       m.MemoryUsedBytes,
		MemoryAvailableBytes:  m.MemoryAvailableBytes,
		LoadAvg_1Min:          m.LoadAvg1Min,
		LoadAvg_5Min:          m.LoadAvg5Min,
		LoadAvg_15Min:         m.LoadAvg15Min,
		DiskTotalBytes:        m.DiskTotalBytes,
		DiskUsedBytes:         m.DiskUsedBytes,
		DiskReadBytesPerSec:   m.DiskReadBytesPerSec,
		DiskWriteBytesPerSec:  m.DiskWriteBytesPerSec,
		NetworkRxBytesPerSec:  m.NetworkRxBytesPerSec,
		NetworkTxBytesPerSec:  m.NetworkTxBytesPerSec,
		CpuTemperatureCelsius: m.CPUTemperatureCelsius,
		UptimeSeconds:         m.UptimeSeconds,
		Timestamp:             m.Timestamp,
	}

	return proto.Marshal(pbMetrics)
}

// shutdown performs graceful shutdown (final flush attempt with timeout)
func (s *Sender) shutdown() {
	log.Println("Flushing pending metrics...")

	// Skip if WAL is disabled
	if s.wal == nil {
		log.Println("✓ Graceful shutdown complete (WAL disabled)")
		return
	}

	// Create context with 5s timeout for final flush
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channel to signal completion
	done := make(chan bool, 1)

	go func() {
		// Try to send pending metrics
		records, err := s.wal.ReadAll()
		if err != nil {
			log.Printf("Failed to read WAL during shutdown: %v", err)
			done <- false
			return
		}

		if len(records) == 0 {
			done <- true
			return
		}

		log.Printf("Sending %d pending metrics...", len(records))

		success := true
		for i, record := range records {
			if err := s.sendRecord(record, false, nil); err != nil {
				log.Printf("Failed to send record %d/%d during shutdown: %v", i+1, len(records), err)
				success = false
				break
			}
		}

		if success {
			if err := s.wal.Clear(); err != nil {
				log.Printf("Warning: Failed to clear WAL: %v", err)
			}
			log.Printf("✓ All pending metrics sent")
		}

		done <- success
	}()

	select {
	case success := <-done:
		if success {
			log.Println("✓ Graceful shutdown complete")
		} else {
			log.Println("⚠ Shutdown completed with errors (metrics preserved in WAL)")
		}
	case <-ctx.Done():
		log.Println("⚠ Shutdown timeout (metrics preserved in WAL)")
	}
}

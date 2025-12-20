package wal

import (
	"context"
	"fmt"
	"log"
	"time"

	"watchflare/client"
	"watchflare/metrics"
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
}

// NewSender creates a new Sender
func NewSender(wal *WAL, grpcClient *client.Client, agentID, agentKey string, metricsIntervalSec int, maxWALSizeMB int) *Sender {
	return &Sender{
		wal:             wal,
		client:          grpcClient,
		agentID:         agentID,
		agentKey:        agentKey,
		metricsInterval: time.Duration(metricsIntervalSec) * time.Second,
		maxWALSize:      int64(maxWALSizeMB) * 1024 * 1024,
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

	// Start ticker
	ticker := time.NewTicker(s.metricsInterval)
	defer ticker.Stop()

	log.Printf("Sender started (interval: %v)", s.metricsInterval)

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

	log.Printf("Replaying %d pending metrics from WAL...", len(records))

	// Try to send all pending records
	success := true
	for i, data := range records {
		if err := s.sendRecord(data); err != nil {
			log.Printf("Failed to send record %d/%d: %v (will retry later)", i+1, len(records), err)
			success = false
			break // Stop at first failure
		}
	}

	// Clear WAL only if all sends succeeded
	if success {
		if err := s.wal.Clear(); err != nil {
			return fmt.Errorf("failed to clear WAL: %w", err)
		}
		log.Printf("✓ All %d pending metrics sent successfully", len(records))
	}

	return nil
}

// collectAndSend collects metrics, persists to WAL, and sends
func (s *Sender) collectAndSend() {
	// 1. Collect metrics
	m, err := metrics.Collect()
	if err != nil {
		log.Printf("Failed to collect metrics: %v", err)
		return
	}

	// If WAL is disabled, send directly
	if s.wal == nil {
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

	// Send all records
	success := true
	for i, record := range records {
		if err := s.sendRecord(record); err != nil {
			log.Printf("Send failed (record %d/%d): %v (will retry in %v)",
				i+1, len(records), err, s.metricsInterval)
			success = false
			break // Stop at first failure
		}
	}

	// Clear WAL only if all sends succeeded
	if success {
		if err := s.wal.Clear(); err != nil {
			log.Printf("Warning: Failed to clear WAL: %v", err)
		} else {
			log.Printf("✓ Metrics sent (CPU: %.1f%%, Mem: %d/%d MB)",
				m.CPUUsagePercent,
				m.MemoryUsedBytes/1024/1024,
				m.MemoryTotalBytes/1024/1024)
		}
	}
}

// sendRecord sends a single serialized metrics record
func (s *Sender) sendRecord(data []byte) error {
	// Deserialize
	var pbMetrics pb.Metrics
	if err := proto.Unmarshal(data, &pbMetrics); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Convert to metrics.SystemMetrics for client
	m := &metrics.SystemMetrics{
		CPUUsagePercent:      pbMetrics.CpuUsagePercent,
		MemoryTotalBytes:     pbMetrics.MemoryTotalBytes,
		MemoryUsedBytes:      pbMetrics.MemoryUsedBytes,
		MemoryAvailableBytes: pbMetrics.MemoryAvailableBytes,
		LoadAvg1Min:          pbMetrics.LoadAvg_1Min,
		LoadAvg5Min:          pbMetrics.LoadAvg_5Min,
		LoadAvg15Min:         pbMetrics.LoadAvg_15Min,
		DiskTotalBytes:       pbMetrics.DiskTotalBytes,
		DiskUsedBytes:        pbMetrics.DiskUsedBytes,
		UptimeSeconds:        pbMetrics.UptimeSeconds,
		Timestamp:            pbMetrics.Timestamp,
	}

	// Send via gRPC
	return s.client.SendMetrics(s.agentID, s.agentKey, m)
}

// serializeMetrics converts SystemMetrics to protobuf bytes
func (s *Sender) serializeMetrics(m *metrics.SystemMetrics) ([]byte, error) {
	pbMetrics := &pb.Metrics{
		CpuUsagePercent:      m.CPUUsagePercent,
		MemoryTotalBytes:     m.MemoryTotalBytes,
		MemoryUsedBytes:      m.MemoryUsedBytes,
		MemoryAvailableBytes: m.MemoryAvailableBytes,
		LoadAvg_1Min:         m.LoadAvg1Min,
		LoadAvg_5Min:         m.LoadAvg5Min,
		LoadAvg_15Min:        m.LoadAvg15Min,
		DiskTotalBytes:       m.DiskTotalBytes,
		DiskUsedBytes:        m.DiskUsedBytes,
		UptimeSeconds:        m.UptimeSeconds,
		Timestamp:            m.Timestamp,
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
			if err := s.sendRecord(record); err != nil {
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

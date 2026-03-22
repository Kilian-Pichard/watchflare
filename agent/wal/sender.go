package wal

import (
	"context"
	"fmt"
	"log/slog"
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
	agentVersion    string
	metricsInterval time.Duration
	maxWALSize      int64
	metricsConfig   *sysinfo.MetricsConfig
}

// NewSender creates a new Sender
func NewSender(wal *WAL, grpcClient *client.Client, agentID, agentKey, agentVersion string, metricsIntervalSec int, maxWALSizeMB int, metricsConfig *sysinfo.MetricsConfig) *Sender {
	return &Sender{
		wal:             wal,
		client:          grpcClient,
		agentID:         agentID,
		agentKey:        agentKey,
		agentVersion:    agentVersion,
		metricsInterval: time.Duration(metricsIntervalSec) * time.Second,
		maxWALSize:      int64(maxWALSizeMB) * 1024 * 1024,
		metricsConfig:   metricsConfig,
	}
}

// Run starts the sender loop (blocks until context cancelled)
func (s *Sender) Run(ctx context.Context) error {
	slog.Info("sender starting")

	// Check WAL size and truncate if needed BEFORE replaying
	if s.wal != nil {
		size, err := s.wal.Size()
		if err != nil {
			slog.Warn("failed to check WAL size on startup", "error", err)
		} else if size > s.maxWALSize {
			slog.Warn("WAL exceeds max size on startup, truncating",
				"size_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024),
				"max_mb", s.maxWALSize/1024/1024)
			if err := s.wal.Truncate(); err != nil {
				slog.Warn("failed to truncate WAL on startup", "error", err)
			} else {
				newSize, _ := s.wal.Size()
				slog.Info("WAL truncated on startup",
					"before_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024),
					"after_mb", fmt.Sprintf("%.2f", float64(newSize)/1024/1024))
			}
		}
	}

	// Replay WAL on startup (after potential truncation)
	if err := s.replayWAL(); err != nil {
		slog.Warn("failed to replay WAL", "error", err)
	}

	// Align first tick to the next wall clock boundary (e.g., :00, :30 for 30s interval)
	now := time.Now()
	intervalSec := int64(s.metricsInterval.Seconds())
	nextBoundary := time.Unix(((now.Unix()/intervalSec)+1)*intervalSec, 0)
	waitDuration := time.Until(nextBoundary)

	slog.Info("aligning to clock boundary", "wait", waitDuration.Round(time.Second).String(), "next_tick", nextBoundary.Format("15:04:05"))

	select {
	case <-time.After(waitDuration):
	case <-ctx.Done():
		slog.Info("sender shutting down before first collection")
		return nil
	}

	ticker := time.NewTicker(s.metricsInterval)
	defer ticker.Stop()

	slog.Info("sender started", "interval", s.metricsInterval)

	// First collection at boundary
	s.collectAndSend()

	for {
		select {
		case <-ticker.C:
			s.collectAndSend()

		case <-ctx.Done():
			slog.Info("sender shutting down")
			s.shutdown()
			return nil
		}
	}
}

// replayWAL sends any pending metrics from WAL on startup
func (s *Sender) replayWAL() error {
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

	slog.Warn("WAL recovery: pending metrics from previous backend downtime", "count", len(records))

	success := true
	for i, data := range records {
		if err := s.sendRecord(data, false, nil); err != nil {
			if errors.IsTimestampError(err) {
				slog.Error("WAL recovery failed: clock out of sync with backend",
					"record", fmt.Sprintf("%d/%d", i+1, len(records)),
					"hint", "ensure system clock is synchronized and restart the agent")
			} else {
				slog.Warn("WAL recovery: failed to send record, will retry",
					"record", fmt.Sprintf("%d/%d", i+1, len(records)),
					"error", err)
			}
			success = false
			break
		}
	}

	if success {
		if err := s.wal.Clear(); err != nil {
			return fmt.Errorf("failed to clear WAL: %w", err)
		}
		slog.Info("WAL recovery complete", "records_sent", len(records))
	}

	return nil
}

// collectAndSend collects metrics, persists to WAL, and sends
func (s *Sender) collectAndSend() {
	m, err := metrics.Collect(s.metricsConfig)
	if err != nil {
		slog.Error("failed to collect metrics", "error", err)
		return
	}

	// Round timestamp to nearest interval boundary
	intervalSec := int64(s.metricsInterval.Seconds())
	m.Timestamp = ((m.Timestamp + intervalSec/2) / intervalSec) * intervalSec

	// Container metrics are sent point-in-time only (not WAL'd)
	containerMetrics := m.ContainerMetrics
	m.ContainerMetrics = nil

	// If WAL is disabled, send directly
	if s.wal == nil {
		m.ContainerMetrics = containerMetrics
		if err := s.client.SendMetrics(s.agentID, s.agentKey, s.agentVersion, m); err != nil {
			slog.Error("send failed, metrics lost (WAL disabled)", "error", err)
		} else {
			slog.Info("metrics sent",
				"cpu_pct", fmt.Sprintf("%.1f", m.CPUUsagePercent),
				"mem_used_mb", m.MemoryUsedBytes/1024/1024,
				"mem_total_mb", m.MemoryTotalBytes/1024/1024)
		}
		return
	}

	data, err := s.serializeMetrics(m)
	if err != nil {
		slog.Error("failed to serialize metrics", "error", err)
		return
	}

	if err := s.wal.Append(data); err != nil {
		slog.Error("failed to append to WAL", "error", err)
		return
	}

	// Check WAL size and truncate if needed (FIFO)
	size, err := s.wal.Size()
	if err != nil {
		slog.Warn("failed to check WAL size", "error", err)
	} else if size > s.maxWALSize {
		slog.Warn("WAL exceeds max size, truncating",
			"size_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024),
			"max_mb", s.maxWALSize/1024/1024)
		if err := s.wal.Truncate(); err != nil {
			slog.Warn("failed to truncate WAL", "error", err)
		} else {
			newSize, _ := s.wal.Size()
			slog.Info("WAL truncated",
				"before_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024),
				"after_mb", fmt.Sprintf("%.2f", float64(newSize)/1024/1024))
		}
	}

	// Send all pending records
	records, err := s.wal.ReadAll()
	if err != nil {
		slog.Error("failed to read WAL", "error", err)
		return
	}

	success := true
	for i, record := range records {
		isLastRecord := i == len(records)-1
		if err := s.sendRecord(record, isLastRecord, containerMetrics); err != nil {
			if errors.IsTimestampError(err) {
				slog.Error("send failed: clock out of sync with backend",
					"record", fmt.Sprintf("%d/%d", i+1, len(records)),
					"retry_in", s.metricsInterval,
					"hint", "ensure system clock is synchronized and restart the agent")
			} else {
				slog.Warn("send failed, will retry",
					"record", fmt.Sprintf("%d/%d", i+1, len(records)),
					"error", err,
					"retry_in", s.metricsInterval)
			}
			success = false
			break
		}
	}

	if success {
		if err := s.wal.Clear(); err != nil {
			slog.Warn("failed to clear WAL", "error", err)
		} else {
			if len(records) > 1 {
				slog.Info("metrics sent (including accumulated during outage)",
					"total_records", len(records),
					"accumulated", len(records)-1)
			} else {
				slog.Info("metrics sent",
					"cpu_pct", fmt.Sprintf("%.1f", m.CPUUsagePercent),
					"mem_used_mb", m.MemoryUsedBytes/1024/1024,
					"mem_total_mb", m.MemoryTotalBytes/1024/1024)
			}
		}
	}
}

// sendRecord sends a single serialized metrics record
func (s *Sender) sendRecord(data []byte, includeContainers bool, containerMetrics []metrics.ContainerMetric) error {
	var pbMetrics pb.Metrics
	if err := proto.Unmarshal(data, &pbMetrics); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

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

	if includeContainers && len(containerMetrics) > 0 {
		m.ContainerMetrics = containerMetrics
	}

	return s.client.SendMetrics(s.agentID, s.agentKey, s.agentVersion, m)
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
	if s.wal == nil {
		slog.Info("graceful shutdown complete (WAL disabled)")
		return
	}

	slog.Info("flushing pending metrics")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool, 1)

	go func() {
		records, err := s.wal.ReadAll()
		if err != nil {
			slog.Error("failed to read WAL during shutdown", "error", err)
			done <- false
			return
		}

		if len(records) == 0 {
			done <- true
			return
		}

		slog.Info("sending pending metrics during shutdown", "count", len(records))

		success := true
		for i, record := range records {
			if err := s.sendRecord(record, false, nil); err != nil {
				slog.Warn("failed to send record during shutdown",
					"record", fmt.Sprintf("%d/%d", i+1, len(records)),
					"error", err)
				success = false
				break
			}
		}

		if success {
			if err := s.wal.Clear(); err != nil {
				slog.Warn("failed to clear WAL during shutdown", "error", err)
			}
		}

		done <- success
	}()

	select {
	case success := <-done:
		if success {
			slog.Info("graceful shutdown complete")
		} else {
			slog.Warn("shutdown completed with errors, metrics preserved in WAL")
		}
	case <-ctx.Done():
		slog.Warn("shutdown timed out, metrics preserved in WAL")
	}
}

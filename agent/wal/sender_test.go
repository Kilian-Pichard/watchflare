package wal

import (
	"testing"
	"time"

	"watchflare-agent/metrics"
	"watchflare-agent/sysinfo"
	pb "watchflare/shared/proto/agent/v1"

	"google.golang.org/protobuf/proto"
)

// newTestSender returns a Sender with nil WAL and client — sufficient for
// testing pure data-transformation methods (serializeMetrics).
func newTestSender() *Sender {
	return NewSender(nil, nil, "agent-id", "agent-key", "1.0.0", 30, 10, &sysinfo.MetricsConfig{})
}

// --- NewSender ---

func TestNewSender_IntervalConversion(t *testing.T) {
	s := NewSender(nil, nil, "id", "key", "v1", 30, 10, &sysinfo.MetricsConfig{})
	if s.metricsInterval != 30*time.Second {
		t.Errorf("metricsInterval = %v, want 30s", s.metricsInterval)
	}
}

func TestNewSender_MaxWALSize(t *testing.T) {
	s := NewSender(nil, nil, "id", "key", "v1", 30, 10, &sysinfo.MetricsConfig{})
	want := int64(10 * 1024 * 1024)
	if s.maxWALSize != want {
		t.Errorf("maxWALSize = %d, want %d", s.maxWALSize, want)
	}
}

// --- serializeMetrics ---

func TestSerializeMetrics_RoundTrip(t *testing.T) {
	s := newTestSender()

	m := &metrics.SystemMetrics{
		CPUUsagePercent:       42.5,
		MemoryTotalBytes:      8 * 1024 * 1024 * 1024,
		MemoryUsedBytes:       4 * 1024 * 1024 * 1024,
		MemoryAvailableBytes:  4 * 1024 * 1024 * 1024,
		LoadAvg1Min:           1.1,
		LoadAvg5Min:           1.5,
		LoadAvg15Min:          2.0,
		DiskTotalBytes:        500 * 1024 * 1024 * 1024,
		DiskUsedBytes:         200 * 1024 * 1024 * 1024,
		DiskReadBytesPerSec:   1024,
		DiskWriteBytesPerSec:  2048,
		NetworkRxBytesPerSec:  512,
		NetworkTxBytesPerSec:  256,
		CPUTemperatureCelsius: 65.0,
		UptimeSeconds:         86400,
		Timestamp:             1711234567,
	}

	data, err := s.serializeMetrics(m)
	if err != nil {
		t.Fatalf("serializeMetrics: %v", err)
	}

	var got pb.Metrics
	if err := proto.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.CpuUsagePercent != m.CPUUsagePercent {
		t.Errorf("CpuUsagePercent = %v, want %v", got.CpuUsagePercent, m.CPUUsagePercent)
	}
	if got.MemoryTotalBytes != m.MemoryTotalBytes {
		t.Errorf("MemoryTotalBytes = %d, want %d", got.MemoryTotalBytes, m.MemoryTotalBytes)
	}
	if got.MemoryUsedBytes != m.MemoryUsedBytes {
		t.Errorf("MemoryUsedBytes = %d, want %d", got.MemoryUsedBytes, m.MemoryUsedBytes)
	}
	if got.DiskTotalBytes != m.DiskTotalBytes {
		t.Errorf("DiskTotalBytes = %d, want %d", got.DiskTotalBytes, m.DiskTotalBytes)
	}
	if got.DiskReadBytesPerSec != m.DiskReadBytesPerSec {
		t.Errorf("DiskReadBytesPerSec = %d, want %d", got.DiskReadBytesPerSec, m.DiskReadBytesPerSec)
	}
	if got.NetworkRxBytesPerSec != m.NetworkRxBytesPerSec {
		t.Errorf("NetworkRxBytesPerSec = %d, want %d", got.NetworkRxBytesPerSec, m.NetworkRxBytesPerSec)
	}
	if got.CpuTemperatureCelsius != m.CPUTemperatureCelsius {
		t.Errorf("CpuTemperatureCelsius = %v, want %v", got.CpuTemperatureCelsius, m.CPUTemperatureCelsius)
	}
	if got.UptimeSeconds != m.UptimeSeconds {
		t.Errorf("UptimeSeconds = %d, want %d", got.UptimeSeconds, m.UptimeSeconds)
	}
	if got.Timestamp != m.Timestamp {
		t.Errorf("Timestamp = %d, want %d", got.Timestamp, m.Timestamp)
	}
	if got.LoadAvg_1Min != m.LoadAvg1Min {
		t.Errorf("LoadAvg1Min = %v, want %v", got.LoadAvg_1Min, m.LoadAvg1Min)
	}
	if got.LoadAvg_5Min != m.LoadAvg5Min {
		t.Errorf("LoadAvg5Min = %v, want %v", got.LoadAvg_5Min, m.LoadAvg5Min)
	}
	if got.LoadAvg_15Min != m.LoadAvg15Min {
		t.Errorf("LoadAvg15Min = %v, want %v", got.LoadAvg_15Min, m.LoadAvg15Min)
	}
	if got.MemoryAvailableBytes != m.MemoryAvailableBytes {
		t.Errorf("MemoryAvailableBytes = %d, want %d", got.MemoryAvailableBytes, m.MemoryAvailableBytes)
	}
	if got.DiskWriteBytesPerSec != m.DiskWriteBytesPerSec {
		t.Errorf("DiskWriteBytesPerSec = %d, want %d", got.DiskWriteBytesPerSec, m.DiskWriteBytesPerSec)
	}
	if got.NetworkTxBytesPerSec != m.NetworkTxBytesPerSec {
		t.Errorf("NetworkTxBytesPerSec = %d, want %d", got.NetworkTxBytesPerSec, m.NetworkTxBytesPerSec)
	}
}

func TestSerializeMetrics_SensorReadings(t *testing.T) {
	s := newTestSender()

	m := &metrics.SystemMetrics{
		SensorReadings: []metrics.SensorReading{
			{Key: "cpu-thermal", TemperatureCelsius: 72.3},
			{Key: "gpu-thermal", TemperatureCelsius: 65.0},
		},
	}

	data, err := s.serializeMetrics(m)
	if err != nil {
		t.Fatalf("serializeMetrics: %v", err)
	}

	var got pb.Metrics
	if err := proto.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.SensorReadings) != 2 {
		t.Fatalf("SensorReadings len = %d, want 2", len(got.SensorReadings))
	}
	if got.SensorReadings[0].Key != "cpu-thermal" {
		t.Errorf("SensorReadings[0].Key = %q, want %q", got.SensorReadings[0].Key, "cpu-thermal")
	}
	if got.SensorReadings[0].TemperatureCelsius != 72.3 {
		t.Errorf("SensorReadings[0].TemperatureCelsius = %v, want 72.3", got.SensorReadings[0].TemperatureCelsius)
	}
	if got.SensorReadings[1].Key != "gpu-thermal" {
		t.Errorf("SensorReadings[1].Key = %q, want %q", got.SensorReadings[1].Key, "gpu-thermal")
	}
}

func TestSerializeMetrics_EmptyMetrics(t *testing.T) {
	s := newTestSender()
	_, err := s.serializeMetrics(&metrics.SystemMetrics{})
	if err != nil {
		t.Errorf("serializeMetrics with zero values: %v", err)
	}
}

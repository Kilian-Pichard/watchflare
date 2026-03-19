package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Event represents a server event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ServerUpdate represents a server status update
type ServerUpdate struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	IPv4Address      string `json:"ip_address_v4,omitempty"`
	IPv6Address      string `json:"ip_address_v6,omitempty"`
	ConfiguredIP     string `json:"configured_ip,omitempty"`
	IgnoreIPMismatch bool   `json:"ignore_ip_mismatch"`
	LastSeen         string `json:"last_seen"`
	Reactivated      bool   `json:"reactivated,omitempty"`      // True if agent was reactivated (UUID reused)
	Hostname         string `json:"hostname,omitempty"`         // Hostname for reactivation notification
	ClockDesync      bool   `json:"clock_desync,omitempty"`     // True if agent's clock is out of sync
}

// MetricsUpdate represents a real-time metrics update
type MetricsUpdate struct {
	ServerID             string  `json:"server_id"`
	Timestamp            string  `json:"timestamp"`
	CPUUsagePercent      float64 `json:"cpu_usage_percent"`
	MemoryTotalBytes     uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes      uint64  `json:"memory_used_bytes"`
	MemoryAvailableBytes uint64  `json:"memory_available_bytes"`
	LoadAvg1Min          float64 `json:"load_avg_1min"`
	LoadAvg5Min          float64 `json:"load_avg_5min"`
	LoadAvg15Min         float64 `json:"load_avg_15min"`
	DiskTotalBytes        uint64  `json:"disk_total_bytes"`
	DiskUsedBytes         uint64  `json:"disk_used_bytes"`
	DiskReadBytesPerSec   uint64  `json:"disk_read_bytes_per_sec"`
	DiskWriteBytesPerSec  uint64  `json:"disk_write_bytes_per_sec"`
	NetworkRxBytesPerSec  uint64  `json:"network_rx_bytes_per_sec"`
	NetworkTxBytesPerSec  uint64  `json:"network_tx_bytes_per_sec"`
	CPUTemperatureCelsius float64 `json:"cpu_temperature_celsius"`
	UptimeSeconds         uint64  `json:"uptime_seconds"`
}

// MetricsUpdateMinified represents a minified metrics update for SSE (reduces bandwidth)
// Format: {"s":"srv1","t":1702741200,"c":22.5,"mu":4294967296,"mt":8589934592,...}
type MetricsUpdateMinified struct {
	ServerID         string  `json:"s"`  // server_id
	Timestamp        int64   `json:"t"`  // Unix timestamp
	CPU              float64 `json:"c"`  // cpu_usage_percent
	MemoryUsed       uint64  `json:"mu"` // memory_used_bytes
	MemoryTotal      uint64  `json:"mt"` // memory_total_bytes
	MemoryAvailable  uint64  `json:"ma"` // memory_available_bytes
	DiskUsed         uint64  `json:"du"` // disk_used_bytes
	DiskTotal        uint64  `json:"dt"` // disk_total_bytes
	LoadAvg1         float64 `json:"l1"`  // load_avg_1min
	LoadAvg5         float64 `json:"l5"`  // load_avg_5min
	LoadAvg15        float64 `json:"l15"` // load_avg_15min
	DiskReadRate     uint64  `json:"dr"`  // disk_read_bytes_per_sec
	DiskWriteRate    uint64  `json:"dw"`  // disk_write_bytes_per_sec
	NetRxRate        uint64  `json:"nr"`  // network_rx_bytes_per_sec
	NetTxRate        uint64  `json:"nt"`  // network_tx_bytes_per_sec
	CPUTemp          float64 `json:"tmp"` // cpu_temperature_celsius
	Uptime           uint64  `json:"u"`   // uptime_seconds
}

// AggregatedMetricsUpdate represents aggregated metrics from all online servers
type AggregatedMetricsUpdate struct {
	Timestamp            string  `json:"timestamp"`
	CPUUsagePercent      float64 `json:"cpu_usage_percent"`
	MemoryTotalBytes     uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes      uint64  `json:"memory_used_bytes"`
	MemoryAvailableBytes uint64  `json:"memory_available_bytes"`
	DiskTotalBytes       uint64  `json:"disk_total_bytes"`
	DiskUsedBytes        uint64  `json:"disk_used_bytes"`
}

// ContainerMetricMinified represents a minified container metric for SSE
type ContainerMetricMinified struct {
	ID   string  `json:"i"`  // container_id (short hash)
	Name string  `json:"n"`  // container_name
	CPU  float64 `json:"c"`  // cpu_percent
	MU   uint64  `json:"mu"` // memory_used_bytes
	ML   uint64  `json:"ml"` // memory_limit_bytes
	NR   uint64  `json:"nr"` // network_rx_bytes_per_sec
	NT   uint64  `json:"nt"` // network_tx_bytes_per_sec
}

// ContainerMetricsUpdate represents container metrics for SSE broadcast
type ContainerMetricsUpdate struct {
	ServerID  string                    `json:"s"`
	Timestamp int64                     `json:"t"`
	Metrics   []ContainerMetricMinified `json:"m"`
}

// Client represents an SSE client connection
type Client struct {
	ID      string
	Channel chan Event
}

// Broker manages SSE client connections and event broadcasting
type Broker struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

var (
	broker *Broker
	once   sync.Once
)

// GetBroker returns the singleton SSE broker instance
func GetBroker() *Broker {
	once.Do(func() {
		broker = &Broker{
			clients: make(map[string]*Client),
		}
	})
	return broker
}

// AddClient registers a new SSE client
func (b *Broker) AddClient(clientID string) *Client {
	b.mu.Lock()
	defer b.mu.Unlock()

	client := &Client{
		ID:      clientID,
		Channel: make(chan Event, 10),
	}

	b.clients[clientID] = client
	log.Printf("SSE client connected: %s (total: %d)", clientID, len(b.clients))

	return client
}

// RemoveClient unregisters an SSE client
func (b *Broker) RemoveClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if client, exists := b.clients[clientID]; exists {
		close(client.Channel)
		delete(b.clients, clientID)
		log.Printf("SSE client disconnected: %s (total: %d)", clientID, len(b.clients))
	}
}

// Broadcast sends an event to all connected clients
func (b *Broker) Broadcast(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, client := range b.clients {
		select {
		case client.Channel <- event:
			// Event sent successfully
		default:
			// Channel is full, skip this client
			log.Printf("Warning: Client %s channel is full, skipping event", client.ID)
		}
	}

	log.Printf("Broadcast event: %s to %d clients", event.Type, len(b.clients))
}

// BroadcastServerUpdate sends a server update event to all clients
func (b *Broker) BroadcastServerUpdate(update ServerUpdate) {
	event := Event{
		Type: "server_update",
		Data: update,
	}
	b.Broadcast(event)
}

// BroadcastMetricsUpdate sends a metrics update event to all clients (minified format)
func (b *Broker) BroadcastMetricsUpdate(update MetricsUpdate) {
	// Convert to minified format for bandwidth optimization
	minified := toMinifiedMetrics(update)

	event := Event{
		Type: "metrics_update",
		Data: minified,
	}
	b.Broadcast(event)
}

// BroadcastAggregatedMetricsUpdate sends an aggregated metrics update event to all clients
func (b *Broker) BroadcastAggregatedMetricsUpdate(update AggregatedMetricsUpdate) {
	event := Event{
		Type: "aggregated_metrics_update",
		Data: update,
	}
	b.Broadcast(event)
}

// BroadcastContainerMetricsUpdate sends container metrics to all SSE clients
func (b *Broker) BroadcastContainerMetricsUpdate(update ContainerMetricsUpdate) {
	event := Event{
		Type: "container_metrics_update",
		Data: update,
	}
	b.Broadcast(event)
}

// toMinifiedMetrics converts MetricsUpdate to minified format
func toMinifiedMetrics(update MetricsUpdate) MetricsUpdateMinified {
	// Parse timestamp string to Unix timestamp
	var timestamp int64
	if t, err := time.Parse(time.RFC3339, update.Timestamp); err == nil {
		timestamp = t.Unix()
	}

	return MetricsUpdateMinified{
		ServerID:        update.ServerID,
		Timestamp:       timestamp,
		CPU:             update.CPUUsagePercent,
		MemoryUsed:      update.MemoryUsedBytes,
		MemoryTotal:     update.MemoryTotalBytes,
		MemoryAvailable: update.MemoryAvailableBytes,
		DiskUsed:        update.DiskUsedBytes,
		DiskTotal:       update.DiskTotalBytes,
		LoadAvg1:        update.LoadAvg1Min,
		LoadAvg5:        update.LoadAvg5Min,
		LoadAvg15:       update.LoadAvg15Min,
		DiskReadRate:    update.DiskReadBytesPerSec,
		DiskWriteRate:   update.DiskWriteBytesPerSec,
		NetRxRate:       update.NetworkRxBytesPerSec,
		NetTxRate:       update.NetworkTxBytesPerSec,
		CPUTemp:         update.CPUTemperatureCelsius,
		Uptime:          update.UptimeSeconds,
	}
}

// FormatSSE formats an event as SSE protocol message
func FormatSSE(event Event) string {
	data, err := json.Marshal(event.Data)
	if err != nil {
		log.Printf("Error marshaling SSE event: %v", err)
		return ""
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}

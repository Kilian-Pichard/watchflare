package cache

import (
	"context"
	"log/slog"
	"time"

	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/sse"

	"gorm.io/gorm"
)

// SyncWorker periodically syncs heartbeat cache to database
type SyncWorker struct {
	cache    *HeartbeatCache
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSyncWorker creates a new sync worker
func NewSyncWorker(interval time.Duration) *SyncWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SyncWorker{
		cache:    GetCache(),
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the sync worker
func (w *SyncWorker) Start() {
	slog.Info("heartbeat sync worker started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.syncToDatabase()

		case <-w.ctx.Done():
			slog.Info("heartbeat sync worker stopped")
			return
		}
	}
}

// Stop stops the sync worker
func (w *SyncWorker) Stop() {
	w.cancel()
}

// syncToDatabase writes updated heartbeat data to the database
func (w *SyncWorker) syncToDatabase() {
	allData := w.cache.GetAll()
	syncCount := 0

	for _, data := range allData {
		if !data.Updated {
			continue // Skip if not updated since last sync
		}

		// Update database
		var server models.Server
		result := database.DB.Where("agent_id = ?", data.AgentID).First(&server)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				slog.Warn("agent not found in database, skipping sync", "agent_id", data.AgentID)
			} else {
				slog.Error("failed to query agent", "agent_id", data.AgentID, "error", result.Error)
			}
			continue
		}

		// Update server status and IP addresses
		updates := map[string]interface{}{
			"last_seen":     data.LastSeen,
			"status":        data.Status,
			"ip_address_v4": data.IPv4Address,
			"ip_address_v6": data.IPv6Address,
		}

		if err := database.DB.Model(&server).Updates(updates).Error; err != nil {
			slog.Error("failed to sync heartbeat", "agent_id", data.AgentID, "error", err)
			continue
		}

		// Mark as synced
		w.cache.MarkSynced(data.AgentID)
		syncCount++
	}

	if syncCount > 0 {
		slog.Info("synced heartbeats to database", "count", syncCount)
	}
}

// StaleChecker periodically checks for stale agents and marks them as offline
type StaleChecker struct {
	cache    *HeartbeatCache
	interval time.Duration
	timeout  time.Duration // How long before an agent is considered offline
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStaleChecker creates a new stale checker
func NewStaleChecker(interval, timeout time.Duration) *StaleChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &StaleChecker{
		cache:    GetCache(),
		interval: interval,
		timeout:  timeout,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the stale checker
func (c *StaleChecker) Start() {
	slog.Info("heartbeat stale checker started", "interval", c.interval, "timeout", c.timeout)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkStaleAgents()

		case <-c.ctx.Done():
			slog.Info("heartbeat stale checker stopped")
			return
		}
	}
}

// Stop stops the stale checker
func (c *StaleChecker) Stop() {
	c.cancel()
}

// checkStaleAgents marks agents as offline if they haven't sent a heartbeat recently
func (c *StaleChecker) checkStaleAgents() {
	staleAgents := c.cache.CheckStale(c.timeout)

	if len(staleAgents) == 0 {
		return
	}

	// For each stale agent, broadcast SSE update and log
	for _, agentID := range staleAgents {
		// Get server details for SSE broadcast
		var server models.Server
		result := database.DB.Where("agent_id = ?", agentID).First(&server)
		if result.Error != nil {
			slog.Warn("stale agent not found in database", "agent_id", agentID)
			continue
		}

		// Get cached data
		data, ok := c.cache.Get(agentID)
		if !ok {
			continue
		}

		// Broadcast offline status to SSE
		broker := sse.GetBroker()
		configuredIP := ""
		if server.ConfiguredIP != nil {
			configuredIP = *server.ConfiguredIP
		}
		broker.BroadcastServerUpdate(sse.ServerUpdate{
			ID:               server.ID,
			Status:           "offline",
			IPv4Address:      data.IPv4Address,
			IPv6Address:      data.IPv6Address,
			ConfiguredIP:     configuredIP,
			IgnoreIPMismatch: server.IgnoreIPMismatch,
			LastSeen:         data.LastSeen.Format(time.RFC3339),
			ClockDesync:      data.ClockDesync,
		})

		slog.Warn("agent marked as offline", "agent_id", agentID, "stale_after", c.timeout)
	}
}

package cache

import (
	"context"
	"log"
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
	log.Printf("Heartbeat sync worker started (interval: %v)", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.syncToDatabase()

		case <-w.ctx.Done():
			log.Println("Heartbeat sync worker stopped")
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
				log.Printf("Warning: Agent %s not found in database, skipping sync", data.AgentID)
			} else {
				log.Printf("Error: Failed to query agent %s: %v", data.AgentID, result.Error)
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
			log.Printf("Error: Failed to sync heartbeat for agent %s: %v", data.AgentID, err)
			continue
		}

		// Mark as synced
		w.cache.MarkSynced(data.AgentID)
		syncCount++
	}

	if syncCount > 0 {
		log.Printf("✓ Synced %d heartbeat(s) to database", syncCount)
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
	log.Printf("Heartbeat stale checker started (interval: %v, timeout: %v)", c.interval, c.timeout)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkStaleAgents()

		case <-c.ctx.Done():
			log.Println("Heartbeat stale checker stopped")
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
			log.Printf("Warning: Stale agent %s not found in database", agentID)
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
		})

		log.Printf("⚠ Agent %s marked as offline (no heartbeat for %v)", agentID, c.timeout)
	}
}

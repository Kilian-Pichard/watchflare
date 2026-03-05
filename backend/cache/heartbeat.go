package cache

import (
	"sync"
	"time"
)

// HeartbeatData represents cached heartbeat information for an agent
type HeartbeatData struct {
	AgentID     string
	LastSeen    time.Time
	Status      string // "online" or "offline"
	IPv4Address string
	IPv6Address string
	Updated     bool // Flag indicating if data has been updated since last DB sync
}

// HeartbeatCache stores heartbeat data in memory
type HeartbeatCache struct {
	mu    sync.RWMutex
	cache map[string]*HeartbeatData // Key: agent_id
}

var (
	globalCache *HeartbeatCache
	once        sync.Once
)

// GetCache returns the global heartbeat cache instance (singleton)
func GetCache() *HeartbeatCache {
	once.Do(func() {
		globalCache = &HeartbeatCache{
			cache: make(map[string]*HeartbeatData),
		}
	})
	return globalCache
}

// Update updates heartbeat data for an agent
func (c *HeartbeatCache) Update(agentID, ipv4, ipv6 string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if existing, ok := c.cache[agentID]; ok {
		// Update existing entry
		existing.LastSeen = now
		existing.Status = "online"
		existing.IPv4Address = ipv4
		existing.IPv6Address = ipv6
		existing.Updated = true
	} else {
		// Create new entry
		c.cache[agentID] = &HeartbeatData{
			AgentID:     agentID,
			LastSeen:    now,
			Status:      "online",
			IPv4Address: ipv4,
			IPv6Address: ipv6,
			Updated:     true,
		}
	}
}

// Get retrieves heartbeat data for an agent
func (c *HeartbeatCache) Get(agentID string) (*HeartbeatData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.cache[agentID]
	if !ok {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &HeartbeatData{
		AgentID:     data.AgentID,
		LastSeen:    data.LastSeen,
		Status:      data.Status,
		IPv4Address: data.IPv4Address,
		IPv6Address: data.IPv6Address,
		Updated:     data.Updated,
	}, true
}

// GetAll returns all cached heartbeat data
func (c *HeartbeatCache) GetAll() []*HeartbeatData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*HeartbeatData, 0, len(c.cache))
	for _, data := range c.cache {
		result = append(result, &HeartbeatData{
			AgentID:     data.AgentID,
			LastSeen:    data.LastSeen,
			Status:      data.Status,
			IPv4Address: data.IPv4Address,
			IPv6Address: data.IPv6Address,
			Updated:     data.Updated,
		})
	}
	return result
}

// MarkSynced marks an agent's heartbeat data as synced to DB
func (c *HeartbeatCache) MarkSynced(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if data, ok := c.cache[agentID]; ok {
		data.Updated = false
	}
}

// CheckStale marks agents as offline if they haven't sent a heartbeat in >15s
func (c *HeartbeatCache) CheckStale(timeout time.Duration) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	staleAgents := []string{}

	for agentID, data := range c.cache {
		if data.Status == "online" && now.Sub(data.LastSeen) > timeout {
			data.Status = "offline"
			data.Updated = true
			staleAgents = append(staleAgents, agentID)
		}
	}

	return staleAgents
}

// Remove removes a specific agent from the cache
func (c *HeartbeatCache) Remove(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, agentID)
}

// Clear removes all cached data (used for testing)
func (c *HeartbeatCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*HeartbeatData)
}

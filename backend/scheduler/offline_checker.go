package scheduler

import (
	"log"
	"time"

	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/sse"
)

const (
	CheckInterval     = 15 * time.Second // Check every 15 seconds
	OfflineThreshold  = 60 * time.Second // Mark offline if no heartbeat for 60 seconds
)

// StartOfflineChecker starts the background task to check for offline servers
func StartOfflineChecker() {
	ticker := time.NewTicker(CheckInterval)

	log.Printf("Starting offline checker (interval: %v, threshold: %v)", CheckInterval, OfflineThreshold)

	go func() {
		for range ticker.C {
			checkOfflineServers()
		}
	}()
}

// checkOfflineServers finds servers that haven't sent heartbeats and marks them offline
func checkOfflineServers() {
	var servers []models.Server

	// Find all servers that are currently "online"
	if err := database.DB.Where("status = ?", "online").Find(&servers).Error; err != nil {
		log.Printf("Error fetching online servers: %v", err)
		return
	}

	now := time.Now()
	offlineCount := 0
	broker := sse.GetBroker()

	for _, server := range servers {
		// Check if last_seen exists and is older than threshold
		if server.LastSeen != nil {
			timeSinceLastSeen := now.Sub(*server.LastSeen)

			if timeSinceLastSeen > OfflineThreshold {
				// Mark server as offline
				if err := database.DB.Model(&server).Update("status", "offline").Error; err != nil {
					log.Printf("Error updating server %s to offline: %v", server.ID, err)
					continue
				}

				log.Printf("Server %s (%s) marked as offline (last seen: %v ago)",
					server.ID, server.Name, timeSinceLastSeen.Round(time.Second))

				// Broadcast SSE event
				var ipv4, ipv6 string
				if server.IPAddressV4 != nil {
					ipv4 = *server.IPAddressV4
				}
				if server.IPAddressV6 != nil {
					ipv6 = *server.IPAddressV6
				}

				broker.BroadcastServerUpdate(sse.ServerUpdate{
					ID:          server.ID,
					Status:      "offline",
					IPv4Address: ipv4,
					IPv6Address: ipv6,
					LastSeen:    server.LastSeen.Format(time.RFC3339),
				})

				offlineCount++
			}
		}
	}

	if offlineCount > 0 {
		log.Printf("Marked %d server(s) as offline", offlineCount)
	}
}

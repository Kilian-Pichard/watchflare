package handlers

import (
	"fmt"
	"watchflare/backend/config"
	"watchflare/backend/sse"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HostEvents handles the global SSE stream — all host events broadcast to all clients.
// GET /api/v1/hosts/events
func HostEvents(c *gin.Context) {
	sseConnect(c, func(clientID string) *sse.Client {
		return sse.GetBroker().AddClient(clientID)
	})
}

// HostDetailEvents handles a per-host SSE stream — only events for the given host are delivered.
// aggregated_metrics_update is never sent on this stream (not meaningful for a single host).
// GET /api/v1/hosts/:id/events
func HostDetailEvents(c *gin.Context) {
	hostID := c.Param("id")
	sseConnect(c, func(clientID string) *sse.Client {
		return sse.GetBroker().AddClientWithHostFilter(clientID, hostID)
	})
}

// sseConnect sets up SSE headers, registers the client via addClient, streams events
// until the request context is cancelled, then removes the client from the broker.
func sseConnect(c *gin.Context, addClient func(clientID string) *sse.Client) {
	// Reflect the request origin only when it is explicitly allowed.
	requestOrigin := c.Request.Header.Get("Origin")
	allowedOrigin := ""
	for _, origin := range config.AppConfig.CORSOrigins {
		if origin == requestOrigin {
			allowedOrigin = origin
			break
		}
	}
	if allowedOrigin != "" {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
	}
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	clientID := uuid.New().String()
	broker := sse.GetBroker()
	client := addClient(clientID)
	defer broker.RemoveClient(clientID)

	// Send initial connection confirmation.
	c.Writer.Write([]byte(sse.FormatSSE(sse.Event{
		Type: sse.EventTypeConnected,
		Data: map[string]string{"client_id": clientID},
	})))
	c.Writer.Flush()

	clientChan := c.Request.Context().Done()
	for {
		select {
		case event := <-client.Channel:
			msg := sse.FormatSSE(event)
			if msg != "" {
				fmt.Fprint(c.Writer, msg)
				c.Writer.Flush()
			}
		case <-clientChan:
			return
		}
	}
}

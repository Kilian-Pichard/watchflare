package handlers

import (
	"fmt"
	"watchflare/backend/config"
	"watchflare/backend/sse"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ServerEvents handles SSE connections for real-time server updates
func ServerEvents(c *gin.Context) {
	// Set CORS headers explicitly for SSE
	// EventSource requires explicit origin when using credentials
	requestOrigin := c.Request.Header.Get("Origin")

	// Check if the request origin is in the allowed list
	allowedOrigin := ""
	for _, origin := range config.AppConfig.CORSOrigins {
		if origin == requestOrigin {
			allowedOrigin = origin
			break
		}
	}

	// Fallback to first allowed origin if request origin not found
	if allowedOrigin == "" && len(config.AppConfig.CORSOrigins) > 0 {
		allowedOrigin = config.AppConfig.CORSOrigins[0]
	}

	c.Header("Access-Control-Allow-Origin", allowedOrigin)
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Generate unique client ID
	clientID := uuid.New().String()

	// Get the SSE broker
	broker := sse.GetBroker()

	// Register the client
	client := broker.AddClient(clientID)
	defer broker.RemoveClient(clientID)

	// Send initial connection message
	c.Writer.Write([]byte(sse.FormatSSE(sse.Event{
		Type: "connected",
		Data: map[string]string{"client_id": clientID},
	})))
	c.Writer.Flush()

	// Stream events to the client
	clientChan := c.Request.Context().Done()
	for {
		select {
		case event := <-client.Channel:
			// Send event to client
			msg := sse.FormatSSE(event)
			if msg != "" {
				fmt.Fprint(c.Writer, msg)
				c.Writer.Flush()
			}

		case <-clientChan:
			// Client disconnected
			return
		}
	}
}

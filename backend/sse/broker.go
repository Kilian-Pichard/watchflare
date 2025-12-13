package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Event represents a server event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ServerUpdate represents a server status update
type ServerUpdate struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	IPv4Address   string `json:"ip_address_v4,omitempty"`
	IPv6Address   string `json:"ip_address_v6,omitempty"`
	LastSeen      string `json:"last_seen"`
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

// FormatSSE formats an event as SSE protocol message
func FormatSSE(event Event) string {
	data, err := json.Marshal(event.Data)
	if err != nil {
		log.Printf("Error marshaling SSE event: %v", err)
		return ""
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}

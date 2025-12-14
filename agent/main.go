package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"watchflare/agent/client"
	"watchflare/agent/config"
	"watchflare/agent/sysinfo"
)

const (
	DefaultServerHost   = "localhost"
	DefaultServerPort   = "50051"
	HeartbeatInterval   = 30 * time.Second
	HeartbeatMaxRetries = 3
)

func main() {
	// Parse command line flags
	token := flag.String("token", "", "Registration token")
	serverHost := flag.String("host", DefaultServerHost, "Backend server host")
	serverPort := flag.String("port", DefaultServerPort, "Backend server port")
	registerOnly := flag.Bool("register-only", false, "Register agent and exit (don't start heartbeat loop)")
	flag.Parse()

	log.Println("Watchflare Agent starting...")

	// Check if already registered
	var cfg *config.Config
	var err error

	if config.Exists() {
		log.Println("Loading existing configuration...")
		cfg, err = config.Load()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		// Registration required
		if *token == "" {
			log.Fatal("Registration token is required for first-time setup. Use --token flag")
		}

		log.Println("Registering agent with backend...")
		cfg, err = register(*token, *serverHost, *serverPort)
		if err != nil {
			log.Fatalf("Registration failed: %v", err)
		}

		log.Println("✅ Registration successful!")
		log.Printf("Agent ID: %s", cfg.AgentID)

		// If register-only mode, exit here
		if *registerOnly {
			log.Println("Registration complete. Exiting (--register-only mode)")
			return
		}
	}

	// Start heartbeat loop
	log.Println("Starting heartbeat loop...")
	if err := runHeartbeatLoop(cfg); err != nil {
		log.Fatalf("Heartbeat loop failed: %v", err)
	}
}

// register performs initial agent registration
func register(token, serverHost, serverPort string) (*config.Config, error) {
	// Collect system information
	info, err := sysinfo.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect system info: %w", err)
	}

	// Connect to backend
	grpcClient, err := client.New(serverHost, serverPort)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backend: %w", err)
	}
	defer grpcClient.Close()

	// Register with backend
	agentID, agentKey, err := grpcClient.Register(
		token,
		info.Hostname,
		info.IPv4Address,
		info.IPv6Address,
		info.Platform,
		info.PlatformVersion,
		info.PlatformFamily,
		info.Architecture,
		info.Kernel,
	)
	if err != nil {
		return nil, err
	}

	// Save configuration
	cfg := &config.Config{
		ServerHost: serverHost,
		ServerPort: serverPort,
		AgentID:    agentID,
		AgentKey:   agentKey,
	}

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return cfg, nil
}

// runHeartbeatLoop sends periodic heartbeats to the backend
func runHeartbeatLoop(cfg *config.Config) error {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	// Connect to backend
	grpcClient, err := client.New(cfg.ServerHost, cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("failed to connect to backend: %w", err)
	}
	defer grpcClient.Close()

	// Send initial heartbeat immediately
	if err := sendHeartbeat(grpcClient, cfg); err != nil {
		log.Printf("Warning: Initial heartbeat failed: %v", err)
	}

	log.Printf("Heartbeat interval: %v", HeartbeatInterval)

	for {
		select {
		case <-ticker.C:
			if err := sendHeartbeat(grpcClient, cfg); err != nil {
				log.Printf("Warning: Heartbeat failed: %v", err)
			}

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			return nil
		}
	}
}

// sendHeartbeat sends a single heartbeat to the backend
func sendHeartbeat(grpcClient *client.Client, cfg *config.Config) error {
	// Get current IP addresses
	info, err := sysinfo.Collect()
	if err != nil {
		return fmt.Errorf("failed to collect system info: %w", err)
	}

	if err := grpcClient.SendHeartbeat(
		cfg.AgentID,
		cfg.AgentKey,
		info.IPv4Address,
		info.IPv6Address,
	); err != nil {
		return err
	}

	log.Println("✓ Heartbeat sent successfully")
	return nil
}

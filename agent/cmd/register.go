package cmd

import (
	"log"
	"os"
	"runtime"

	"watchflare-agent/client"
	"watchflare-agent/config"
	"watchflare-agent/sysinfo"
	"watchflare-agent/uuid"
)

// AgentVersion is set by main.go from the build-time Version variable
var AgentVersion = "dev"

// Register handles agent registration with the backend
// Returns true if the agent was reactivated (UUID reused), false if new registration
func Register() bool {
	log.Println("Watchflare Agent Registration")
	log.Println("==============================")

	// Parse command line arguments (supports both --flag=value and --flag value)
	var token, host, port string
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case len(arg) > 8 && arg[:8] == "--token=":
			token = arg[8:]
		case arg == "--token" && i+1 < len(os.Args):
			i++
			token = os.Args[i]
		case len(arg) > 7 && arg[:7] == "--host=":
			host = arg[7:]
		case arg == "--host" && i+1 < len(os.Args):
			i++
			host = os.Args[i]
		case len(arg) > 7 && arg[:7] == "--port=":
			port = arg[7:]
		case arg == "--port" && i+1 < len(os.Args):
			i++
			port = os.Args[i]
		}
	}

	// Validate required arguments
	if token == "" {
		log.Fatal("Error: --token is required\nUsage: watchflare-agent register --token=TOKEN [--host=HOST] [--port=PORT]")
	}

	// Set defaults
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "50051"
	}

	log.Printf("Backend: %s:%s", host, port)

	// Collect system information
	log.Println("Collecting system information...")
	info, err := sysinfo.Collect()
	if err != nil {
		log.Fatalf("Failed to collect system info: %v", err)
	}

	log.Printf("  Hostname: %s", info.Hostname)
	log.Printf("  Platform: %s %s", info.Platform, info.PlatformVersion)
	log.Printf("  Architecture: %s", info.Architecture)
	log.Printf("  IPv4: %s", info.IPv4Address)
	if info.IPv6Address != "" {
		log.Printf("  IPv6: %s", info.IPv6Address)
	}

	// Connect to backend with permissive TLS for bootstrap
	log.Println("\nConnecting to backend...")
	grpcClient, err := client.NewForRegistration(host, port)
	if err != nil {
		log.Fatalf("Failed to connect to backend: %v", err)
	}
	defer grpcClient.Close()

	// Detect environment type
	log.Println("Detecting environment...")
	env := sysinfo.DetectEnvironment()
	log.Printf("Environment detected: %s", env.String())

	// Check for existing UUID (for re-registration)
	existingUUID, err := uuid.Load()
	if err != nil {
		log.Printf("Warning: Failed to load existing UUID: %v", err)
		existingUUID = "" // Continue as new registration
	}
	if existingUUID != "" {
		log.Printf("Found existing agent UUID: %s (will reactivate if still valid)", existingUUID)
	}

	// Register with backend
	log.Println("Registering agent...")
	regResp, err := grpcClient.Register(
		token,
		info.Hostname,
		info.IPv4Address,
		info.IPv6Address,
		info.Platform,
		info.PlatformVersion,
		info.PlatformFamily,
		info.Architecture,
		info.Kernel,
		string(env.Type),
		env.Hypervisor,
		env.ContainerRuntime,
		existingUUID,
		AgentVersion,
	)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	// Save CA certificate to disk
	caCertPath := config.GetConfigDir() + "/ca.pem"
	log.Printf("Saving CA certificate to %s...", caCertPath)
	if err := client.SaveCACertificate(regResp.CACert, caCertPath); err != nil {
		log.Fatalf("Failed to save CA certificate: %v", err)
	}

	// Create configuration
	cfg := &config.Config{
		ServerHost: host,
		ServerPort: port,
		AgentID:    regResp.AgentID,
		AgentKey:   regResp.AgentKey,
		CACertFile: caCertPath,
		ServerName: regResp.ServerName,
	}
	cfg.SetDefaults()

	// Save configuration
	log.Println("Saving configuration...")
	if err := config.Save(cfg); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	// Save agent UUID for future re-registrations
	log.Println("Saving agent UUID...")
	if err := uuid.Save(regResp.AgentID); err != nil {
		log.Printf("Warning: Failed to save UUID: %v", err)
		// Not fatal - agent will work, but will create new UUID on next registration
	}

	log.Println("\n✅ Registration successful!")
	if regResp.Reactivated {
		log.Println("⚠️  NOTICE: This agent was merged with an existing agent in the system")
		log.Println("   Reason: Agent UUID was found on disk (/var/lib/watchflare/agent.uuid)")
		log.Println("   This is the same physical server reconnecting, so the existing agent was reactivated")
		log.Println("   If you intended to create a NEW agent, uninstall with data cleanup first")
	}
	log.Printf("Agent ID: %s", regResp.AgentID)
	log.Printf("Config saved to: %s", config.GetConfigDir()+"/"+config.ConfigFile)
	log.Printf("TLS enabled with server: %s", regResp.ServerName)
	if isInstalledViaBrew() {
		log.Println("\nYou can now start the agent with: brew services start watchflare-agent")
	} else if runtime.GOOS == "linux" {
		log.Println("\nYou can now start the agent with: sudo systemctl enable --now watchflare-agent")
	} else {
		log.Println("\nYou can now start the agent with: ./watchflare-agent")
	}

	return regResp.Reactivated
}

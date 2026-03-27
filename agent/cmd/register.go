package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	fmt.Println("Watchflare Agent Registration")
	fmt.Println("==============================")

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
		fmt.Fprintln(os.Stderr, "error: --token is required\nUsage: watchflare-agent register --token=TOKEN [--host=HOST] [--port=PORT]")
		os.Exit(1)
	}

	// Set defaults
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "50051"
	}

	// Collect system information
	slog.Info("collecting system information")
	info, err := sysinfo.Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to collect system info: %v\n", err)
		os.Exit(1)
	}

	slog.Info("system info",
		"hostname", info.Hostname,
		"platform", info.Platform+" "+info.PlatformVersion,
		"arch", info.Architecture,
		"ipv4", info.IPv4Address)
	if info.IPv6Address != "" {
		slog.Info("IPv6 detected", "ipv6", info.IPv6Address)
	}

	// Connect to backend with permissive TLS for bootstrap
	slog.Info("connecting to backend", "host", host, "port", port)
	grpcClient, err := client.NewForRegistration(host, port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to backend: %v\n", err)
		os.Exit(1)
	}
	defer grpcClient.Close()

	// Detect environment type
	env := sysinfo.DetectEnvironment()
	slog.Info("environment detected", "type", env.String())

	// Check for existing UUID (for re-registration)
	existingUUID, err := uuid.Load()
	if err != nil {
		slog.Warn("failed to load existing UUID", "error", err)
		existingUUID = ""
	}
	if existingUUID != "" {
		slog.Info("found existing agent UUID, will reactivate if still valid", "agent_id", existingUUID)
	}

	// Register with backend
	slog.Info("registering agent")
	regResp, err := grpcClient.Register(client.RegisterRequest{
		Token:            token,
		Hostname:         info.Hostname,
		IPv4:             info.IPv4Address,
		IPv6:             info.IPv6Address,
		Platform:         info.Platform,
		PlatformVersion:  info.PlatformVersion,
		PlatformFamily:   info.PlatformFamily,
		Architecture:     info.Architecture,
		Kernel:           info.Kernel,
		EnvironmentType:  string(env.Type),
		Hypervisor:       env.Hypervisor,
		ContainerRuntime: env.ContainerRuntime,
		ExistingUUID:     existingUUID,
		AgentVersion:     AgentVersion,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: registration failed: %v\n", err)
		os.Exit(1)
	}

	// Save CA certificate to disk
	caCertPath := filepath.Join(config.GetConfigDir(), "ca.pem")
	slog.Info("saving CA certificate", "path", caCertPath)
	if err := client.SaveCACertificate(regResp.CACert, caCertPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to save CA certificate: %v\n", err)
		os.Exit(1)
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
	slog.Info("saving configuration")
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to save config: %v\n", err)
		os.Exit(1)
	}

	// Save agent UUID for future re-registrations
	slog.Info("saving agent UUID")
	if err := uuid.Save(regResp.AgentID); err != nil {
		slog.Warn("failed to save UUID", "error", err)
		// Not fatal - agent will work, but will create new UUID on next registration
	}

	fmt.Println()
	fmt.Println("✅ Registration successful!")
	if regResp.Reactivated {
		fmt.Println("⚠️  NOTICE: This agent was merged with an existing agent in the system")
		fmt.Println("   Reason: Agent UUID was found on disk (/var/lib/watchflare/agent.uuid)")
		fmt.Println("   This is the same physical server reconnecting, so the existing agent was reactivated")
		fmt.Println("   If you intended to create a NEW agent, uninstall with data cleanup first")
	}
	slog.Info("agent registered",
		"agent_id", regResp.AgentID,
		"config", config.GetConfigDir()+"/"+config.ConfigFile,
		"tls_server", regResp.ServerName)

	if isInstalledViaBrew() {
		fmt.Println("\nYou can now start the agent with: brew services start watchflare-agent")
	} else if runtime.GOOS == "linux" {
		fmt.Println("\nYou can now start the agent with: sudo systemctl enable --now watchflare-agent")
	} else {
		fmt.Println("\nYou can now start the agent with: ./watchflare-agent")
	}

	return regResp.Reactivated
}

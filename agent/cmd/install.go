package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"watchflare-agent/install"
)

// Install handles agent installation
func Install() {
	log.SetFlags(0) // Remove timestamp for cleaner output

	fmt.Println("=== Watchflare Agent Installation ===")
	fmt.Println()

	// Step 0: Check if running as root
	fmt.Println("[1/7] Checking permissions...")
	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("  → Running as root")

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

	// Get service manager
	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Check for existing installation
	if svcMgr.IsInstalled() {
		fmt.Println("  → Found existing installation")
		if svcMgr.IsRunning() {
			fmt.Println("  → Stopping existing service...")
			if err := svcMgr.Stop(); err != nil {
				log.Printf("Warning: failed to stop service: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
	}

	// Step 1: Create system user
	fmt.Println("\n[2/7] Creating system user...")
	if err := install.CreateUser(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 2: Create directories
	fmt.Println("\n[3/7] Creating directories...")
	if err := install.CreateDirectories(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 3: Install binary
	fmt.Println("\n[4/7] Installing binary...")

	// Get path to current executable (should be in /tmp from bootstrap script)
	binaryPath, err := install.GetBinaryPath()
	if err != nil {
		log.Fatalf("Error: failed to get binary path: %v", err)
	}

	if err := install.InstallBinary(binaryPath); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create log file
	if err := install.CreateLogFile(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 4: Install service
	fmt.Println("\n[5/7] Installing service...")
	if err := svcMgr.Install(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 5: Registration
	fmt.Println("\n[6/7] Agent registration...")
	needsRegistration := true

	// Check if config already exists
	configPath := install.ConfigDir + "/agent.conf"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("  → Configuration file already exists")
		needsRegistration = false
	} else if token != "" {
		// Registration parameters provided
		fmt.Println("  → Registering agent with backend...")

		// Set defaults
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "50051"
		}

		// Save current args
		oldArgs := os.Args

		// Set args for register command
		os.Args = []string{
			os.Args[0],
			"register",
			"--token=" + token,
			"--host=" + host,
			"--port=" + port,
		}

		// Call register
		wasReactivated := Register()

		// Restore args
		os.Args = oldArgs

		needsRegistration = false

		// Show appropriate message
		if wasReactivated {
			fmt.Println("  → Registration successful (existing agent reactivated)")
			fmt.Println("  ⚠️  NOTICE: Agent UUID was found on disk - merged with existing agent")
		} else {
			fmt.Println("  → Registration successful")
		}
	} else {
		fmt.Println("  ⚠ No configuration file found")
		fmt.Printf("  → To register now, run:\n")
		fmt.Printf("     sudo %s/watchflare-agent register --token=YOUR_TOKEN --host=YOUR_HOST\n", install.InstallDir)
	}

	// Step 6: Enable and start service
	fmt.Println("\n[7/7] Starting service...")
	if !needsRegistration {
		// Enable service
		if err := svcMgr.Enable(); err != nil {
			log.Printf("Warning: %v", err)
		}

		// Start service
		if err := svcMgr.Start(); err != nil {
			log.Fatalf("Error: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Check if running
		if svcMgr.IsRunning() {
			fmt.Println("  → Service started successfully")

			// Wait for a few heartbeats then check for clock sync errors
			fmt.Print("  → Checking agent health...")
			time.Sleep(8 * time.Second)
			logContent, err := os.ReadFile("/var/log/watchflare-agent.log")
			if err == nil && strings.Contains(string(logContent), "CLOCK SYNC ERROR") {
				fmt.Println(" ⚠")
				fmt.Println()
				fmt.Println("  ⚠ WARNING: Clock synchronization error detected!")
				fmt.Println("  The system clock is out of sync with the backend (>5min difference).")
				fmt.Println("  Ensure the system clock is synchronized and restart the agent.")
			} else {
				fmt.Println(" ✓")
			}
		} else {
			fmt.Println("  → Service failed to start")
			fmt.Println("  → Check logs: tail -f /var/log/watchflare-agent.log")
		}
	} else {
		fmt.Println("  → Skipped (needs registration first)")
	}

	// Summary
	fmt.Println("\n=== Installation Complete ===")
	fmt.Println()
	fmt.Println("Installation paths:")
	fmt.Printf("  Binary:        %s/watchflare-agent\n", install.InstallDir)
	fmt.Printf("  Configuration: %s/\n", install.ConfigDir)
	fmt.Printf("  Data:          %s/\n", install.DataDir)
	fmt.Println("  Logs:          /var/log/watchflare-agent.log")
	fmt.Println()

	if needsRegistration {
		fmt.Println("Next steps:")
		fmt.Println("  1. Register the agent:")
		fmt.Printf("     sudo %s/watchflare-agent register --token=YOUR_TOKEN --host=YOUR_HOST\n", install.InstallDir)
		fmt.Println()
		fmt.Println("  2. Start the service:")
		if runtime.GOOS == "darwin" {
			fmt.Println("     sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist")
		} else {
			fmt.Println("     sudo systemctl enable watchflare-agent")
			fmt.Println("     sudo systemctl start watchflare-agent")
		}
		fmt.Println()
	} else {
		if token != "" {
			fmt.Println("Registration details:")
			fmt.Printf("  Backend: %s:%s\n", host, port)
			fmt.Println()
		}

		fmt.Println("Service management:")
		if runtime.GOOS == "darwin" {
			fmt.Println("  Status:  sudo launchctl print system/io.watchflare.agent")
			fmt.Println("  Stop:    sudo launchctl bootout system/io.watchflare.agent")
			fmt.Println("  Start:   sudo launchctl bootstrap system /Library/LaunchDaemons/io.watchflare.agent.plist")
			fmt.Println("  Logs:    tail -f /var/log/watchflare-agent.log")
		} else {
			fmt.Println("  Status:  sudo systemctl status watchflare-agent")
			fmt.Println("  Stop:    sudo systemctl stop watchflare-agent")
			fmt.Println("  Start:   sudo systemctl start watchflare-agent")
			fmt.Println("  Restart: sudo systemctl restart watchflare-agent")
			fmt.Println("  Logs:    tail -f /var/log/watchflare-agent.log")
		}
		fmt.Println()
	}

	fmt.Println("Installation successful!")
}

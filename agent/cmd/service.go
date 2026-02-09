package cmd

import (
	"fmt"
	"log"
	"watchflare-agent/install"
)

// Status displays the current status of the agent service
func Status() {
	log.SetFlags(0) // Remove timestamp for cleaner output

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("=== Watchflare Agent Status ===")
	fmt.Println()

	// Check if installed
	if !svcMgr.IsInstalled() {
		fmt.Println("Status: Not installed")
		fmt.Println()
		fmt.Println("To install the agent, run:")
		fmt.Println("  sudo watchflare-agent install --token=YOUR_TOKEN")
		return
	}

	fmt.Println("Installation: ✓ Installed")

	// Check if running
	if svcMgr.IsRunning() {
		fmt.Println("Status:       ✓ Running")
	} else {
		fmt.Println("Status:       ✗ Stopped")
	}

	fmt.Println()
	fmt.Println("Paths:")
	fmt.Printf("  Binary:        %s/watchflare-agent\n", install.InstallDir)
	fmt.Printf("  Configuration: %s/\n", install.ConfigDir)
	fmt.Printf("  Data:          %s/\n", install.DataDir)
	fmt.Println("  Logs:          /var/log/watchflare-agent.log")
}

// StartService starts the agent service
func StartService() {
	log.SetFlags(0)

	// Check root
	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !svcMgr.IsInstalled() {
		log.Fatal("Error: Agent is not installed. Run 'sudo watchflare-agent install' first.")
	}

	if svcMgr.IsRunning() {
		fmt.Println("Agent is already running")
		return
	}

	if err := svcMgr.Start(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Agent started successfully")
}

// StopService stops the agent service
func StopService() {
	log.SetFlags(0)

	// Check root
	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !svcMgr.IsInstalled() {
		log.Fatal("Error: Agent is not installed")
	}

	if !svcMgr.IsRunning() {
		fmt.Println("Agent is already stopped")
		return
	}

	if err := svcMgr.Stop(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Agent stopped successfully")
}

// RestartService restarts the agent service
func RestartService() {
	log.SetFlags(0)

	// Check root
	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !svcMgr.IsInstalled() {
		log.Fatal("Error: Agent is not installed")
	}

	if err := svcMgr.Restart(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Agent restarted successfully")
}

// Logs displays and follows the agent logs
func Logs() {
	log.SetFlags(0)

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !svcMgr.IsInstalled() {
		log.Fatal("Error: Agent is not installed")
	}

	if err := svcMgr.ShowLogs(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

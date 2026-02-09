package cmd

import (
	"fmt"
	"log"

	"watchflare-agent/install"
)

// Uninstall handles agent uninstallation
func Uninstall() {
	log.SetFlags(0) // Remove timestamp for cleaner output

	fmt.Println("=== Watchflare Agent Uninstallation ===")
	fmt.Println()

	// Step 0: Check if running as root
	fmt.Println("[1/5] Checking permissions...")
	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("  → Running as root")

	// Get service manager
	svcMgr, err := install.GetServiceManager()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Step 1: Stop and remove service
	fmt.Println("\n[2/5] Removing service...")
	if svcMgr.IsInstalled() {
		if err := svcMgr.Uninstall(); err != nil {
			log.Printf("Warning: %v", err)
		}
	} else {
		fmt.Println("  → Service not installed")
	}

	// Step 2: Remove binary
	fmt.Println("\n[3/5] Removing binary...")
	if err := install.RemoveFiles(); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Step 3: Ask about data directory
	fmt.Println("\n[4/6] Data and configuration...")
	removeData := install.AskConfirmation("Remove data directory (/var/lib/watchflare)?")
	removeConfig := install.AskConfirmation("Remove configuration directory (/etc/watchflare)?")

	if removeData || removeConfig {
		if err := install.RemoveDirectories(removeData, removeConfig); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Step 4: Ask about logs
	fmt.Println("\n[5/6] Log files...")
	removeLogs := install.AskConfirmation("Remove log file (/var/log/watchflare-agent.log)?")

	if removeLogs {
		if err := install.RemoveLogFile(); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Step 5: Ask about user
	fmt.Println("\n[6/6] System user...")
	removeUser := install.AskConfirmation("Remove system user 'watchflare'?")

	if removeUser {
		if err := install.RemoveUser(); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Summary
	fmt.Println("\n=== Uninstallation Complete ===")
	fmt.Println()

	if !removeData {
		fmt.Println("Note: Data directory preserved at /var/lib/watchflare")
	}
	if !removeConfig {
		fmt.Println("Note: Configuration preserved at /etc/watchflare")
	}
	if !removeLogs {
		fmt.Println("Note: Log file preserved at /var/log/watchflare-agent.log")
	}
	if !removeUser {
		fmt.Println("Note: System user 'watchflare' preserved")
	}

	fmt.Println()
	fmt.Println("Uninstallation successful!")
}

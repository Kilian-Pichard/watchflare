package cmd

import (
	"fmt"
	"os"

	"watchflare-agent/install"
)

// Uninstall handles agent uninstallation
func Uninstall() {
	fmt.Println("=== Watchflare Agent Uninstallation ===")
	fmt.Println()

	fmt.Println("[1/6] Checking permissions...")
	if err := install.CheckRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  → Running as root")

	svcMgr, err := install.GetServiceManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n[2/6] Removing service...")
	if svcMgr.IsInstalled() {
		if err := svcMgr.Uninstall(); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	} else {
		fmt.Println("  → Service not installed")
	}

	fmt.Println("\n[3/6] Removing binary...")
	if err := install.RemoveFiles(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	fmt.Println("\n[4/6] Data and configuration...")
	removeData := install.AskConfirmation(fmt.Sprintf("Remove data directory (%s)?", install.DataDir))
	removeConfig := install.AskConfirmation(fmt.Sprintf("Remove configuration directory (%s)?", install.ConfigDir))

	if removeData || removeConfig {
		if err := install.RemoveDirectories(removeData, removeConfig); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	fmt.Println("\n[5/6] Log files...")
	removeLogs := install.AskConfirmation(fmt.Sprintf("Remove log file (%s)?", install.LogPath))

	if removeLogs {
		if err := install.RemoveLogFile(); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	fmt.Println("\n[6/6] System user...")
	removeUser := install.AskConfirmation("Remove system user 'watchflare'?")

	if removeUser {
		if err := install.RemoveUser(); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	fmt.Println("\n=== Uninstallation Complete ===")
	fmt.Println()

	if !removeData {
		fmt.Printf("Note: Data directory preserved at %s\n", install.DataDir)
	}
	if !removeConfig {
		fmt.Printf("Note: Configuration preserved at %s\n", install.ConfigDir)
	}
	if !removeLogs {
		fmt.Printf("Note: Log file preserved at %s\n", install.LogPath)
	}
	if !removeUser {
		fmt.Println("Note: System user 'watchflare' preserved")
	}

	fmt.Println()
	fmt.Println("Uninstallation successful!")
}

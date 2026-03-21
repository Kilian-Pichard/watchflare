package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"watchflare-agent/install"
	"watchflare-agent/update"
)

// Update handles the `watchflare-agent update [--check]` command.
// It runs in two phases:
//
//	Phase 1 (from /usr/local/bin): download, verify, extract, then re-exec
//	                               a temp copy of itself for Phase 2.
//	Phase 2 (from /tmp):           stop service, replace binary, start service.
func Update() {
	log.SetFlags(0)

	// Detect Phase 2 — internal flags set by Phase 1 via syscall.Exec
	var applyPath, updaterPath, applyVersion string
	checkOnly := false

	for _, arg := range os.Args[2:] {
		switch {
		case strings.HasPrefix(arg, update.ApplyFlag):
			applyPath = strings.TrimPrefix(arg, update.ApplyFlag)
		case strings.HasPrefix(arg, update.UpdaterFlag):
			updaterPath = strings.TrimPrefix(arg, update.UpdaterFlag)
		case strings.HasPrefix(arg, update.VersionFlag):
			applyVersion = strings.TrimPrefix(arg, update.VersionFlag)
		case arg == "--check":
			checkOnly = true
		}
	}

	// Phase 2: apply the already-downloaded and verified binary
	if applyPath != "" {
		if err := update.ApplyExtracted(applyPath, updaterPath); err != nil {
			log.Fatalf("Update failed: %v", err)
		}
		if applyVersion != "" {
			fmt.Printf("✓ Updated to v%s\n", applyVersion)
		}
		fmt.Println("✓ Service restarted")
		return
	}

	// Phase 1: normal update flow
	fmt.Printf("Watchflare Agent v%s\n\n", AgentVersion)

	if AgentVersion == "dev" {
		fmt.Println("Running in dev mode — update checks disabled")
		return
	}

	fmt.Println("Checking for updates...")
	info, err := update.CheckForUpdate(AgentVersion)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !info.UpdateAvailable {
		fmt.Printf("Already up to date (v%s)\n", info.CurrentVersion)
		return
	}

	fmt.Printf("Update available: v%s → v%s\n", info.CurrentVersion, info.LatestVersion)

	if checkOnly {
		fmt.Println()
		fmt.Println("Run the following to upgrade:")
		fmt.Println("  sudo watchflare-agent update")
		return
	}

	// On macOS, updates are managed via Homebrew
	if runtime.GOOS == "darwin" {
		fmt.Println()
		if isInstalledViaBrew() {
			fmt.Println("Updates on macOS are managed via Homebrew:")
			fmt.Println("  brew upgrade watchflare-agent && brew services restart watchflare-agent")
		} else {
			fmt.Println("To update on macOS, install via Homebrew:")
			fmt.Println("  brew tap Kilian-Pichard/watchflare")
			fmt.Println("  brew install watchflare-agent")
		}
		return
	}

	if err := install.CheckRoot(); err != nil {
		log.Fatalf("Error: %v\nHint: run with sudo to apply the update", err)
	}

	fmt.Printf("\nDownloading v%s...\n", info.LatestVersion)
	if err := update.ApplyUpdate(info); err != nil {
		log.Fatalf("Update failed: %v", err)
	}
}

// isInstalledViaBrew returns true if the agent binary is managed by Homebrew
func isInstalledViaBrew() bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	if strings.Contains(self, "/homebrew/") || strings.Contains(self, "/Cellar/") {
		return true
	}
	// Check if brew knows about the package
	cmd := exec.Command("brew", "list", "--formula", "watchflare-agent")
	return cmd.Run() == nil
}

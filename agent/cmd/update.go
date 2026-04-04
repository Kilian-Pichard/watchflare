package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"watchflare-agent/install"
	"watchflare-agent/update"
)

const brewCheckTimeout = 10 * time.Second

// Update handles the `watchflare-agent update [--check]` command.
// It runs in two phases:
//
//	Phase 1 (from /usr/local/bin): download, verify, extract, then re-exec
//	                               a temp copy of itself for Phase 2.
//	Phase 2 (from /tmp):           stop service, replace binary, start service.
func Update() {
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
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown flag %q\nUsage: watchflare-agent update [--check]\n", arg)
			os.Exit(1)
		}
	}

	// Phase 2: apply the already-downloaded and verified binary
	if applyPath != "" {
		if err := update.ApplyExtracted(applyPath, updaterPath); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error: %v\nHint: run with sudo to apply the update\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDownloading v%s...\n", info.LatestVersion)
	if err := update.ApplyUpdate(info); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}
}

// isInstalledViaBrew returns true if the agent binary is managed by Homebrew
func isInstalledViaBrew() bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	if isBrewPath(self) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), brewCheckTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "brew", "list", "--formula", "watchflare-agent")
	return cmd.Run() == nil
}

// isBrewPath returns true if the executable path indicates a Homebrew-managed binary.
// Covers Apple Silicon (/opt/homebrew/) and Intel (/usr/local/Cellar/).
func isBrewPath(path string) bool {
	return strings.Contains(path, "/homebrew/") || strings.Contains(path, "/Cellar/")
}

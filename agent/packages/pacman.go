package packages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const pacmanTimeout = 60 * time.Second

// PacmanCollector collects packages from pacman (Arch Linux)
type PacmanCollector struct {
	pacmanPath string
}

// Name returns the collector name
func (p *PacmanCollector) Name() string {
	return "pacman"
}

// IsAvailable checks if pacman is available
func (p *PacmanCollector) IsAvailable() bool {
	path, err := exec.LookPath("pacman")
	if err != nil {
		return false
	}
	p.pacmanPath = path
	return true
}

// Collect gathers all installed packages from pacman.
// Uses "pacman -Qi" to fetch all package details in a single invocation.
func (p *PacmanCollector) Collect() ([]*Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pacmanTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.pacmanPath, "-Qi")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pacman -Qi failed: %w", err)
	}

	return parsePacmanOutput(output), nil
}

// parsePacmanOutput parses the output of "pacman -Qi" (all packages).
// Blocks are separated by blank lines; each block describes one package.
func parsePacmanOutput(output []byte) []*Package {
	var packages []*Package
	scanner := bufio.NewScanner(bytes.NewReader(output))

	block := make(map[string]string)
	flush := func() {
		if len(block) == 0 {
			return
		}
		if pkg := pacmanBlockToPackage(block); pkg != nil {
			packages = append(packages, pkg)
		}
		block = make(map[string]string)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		block[key] = val
	}
	flush()

	return packages
}

// pacmanBlockToPackage converts a parsed key-value block into a Package.
func pacmanBlockToPackage(block map[string]string) *Package {
	name := block["Name"]
	version := block["Version"]
	if name == "" || version == "" {
		return nil
	}

	var installedAt time.Time
	if dateStr := block["Install Date"]; dateStr != "" {
		if t, err := time.Parse("Mon 02 Jan 2006 03:04:05 PM MST", dateStr); err == nil {
			installedAt = t
		}
	}

	return &Package{
		Name:           name,
		Version:        version,
		Architecture:   block["Architecture"],
		PackageManager: "pacman",
		Source:         block["Repository"],
		InstalledAt:    installedAt,
		PackageSize:    parsePacmanSize(block["Installed Size"]),
		Description:    TruncateDescription(block["Description"]),
	}
}

// parsePacmanSize converts pacman size strings to bytes (e.g. "1.23 MiB").
func parsePacmanSize(sizeStr string) int64 {
	parts := strings.Fields(sizeStr)
	if len(parts) != 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	var multiplier int64
	switch strings.ToLower(parts[1]) {
	case "b":
		multiplier = 1
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0
	}

	return int64(value * float64(multiplier))
}

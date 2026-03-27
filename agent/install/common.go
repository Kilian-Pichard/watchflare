package install

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
)

const (
	// Common paths
	InstallDir = "/usr/local/bin"
	ConfigDir  = "/etc/watchflare"
	DataDir    = "/var/lib/watchflare"
	LogPath    = "/var/log/watchflare-agent.log"
	BinaryName = "watchflare-agent"
)

// ServiceManager defines the interface for OS-specific service management
type ServiceManager interface {
	// Install installs the service (systemd/launchd)
	Install() error

	// Uninstall removes the service
	Uninstall() error

	// Start starts the service
	Start() error

	// Stop stops the service
	Stop() error

	// Restart restarts the service
	Restart() error

	// Enable enables the service to start on boot
	Enable() error

	// IsInstalled checks if the service is installed
	IsInstalled() bool

	// IsRunning checks if the service is running
	IsRunning() bool

	// ShowLogs displays service logs (follows them)
	ShowLogs() error
}

// GetServiceManager is defined in platform-specific files (common_linux.go, common_darwin.go)

// CheckRoot verifies that the program is running as root
func CheckRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run as root (use sudo)")
	}
	return nil
}

// getUserID returns the UID for a username
func getUserID(username string) (int, error) {
	// Use 'id' command which works reliably on both macOS and Linux
	cmd := exec.Command("id", "-u", username)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("user not found or incomplete: %w", err)
	}

	// Parse output (just a number like "200\n")
	var uid int
	_, err = fmt.Sscanf(string(output), "%d", &uid)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UID: %w", err)
	}
	return uid, nil
}

// getGroupID returns the GID for a group name
func getGroupID(groupname string) (int, error) {
	// Standard library works fine for groups on both platforms
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return 0, err
	}
	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, err
	}
	return gid, nil
}

// CreateUser creates the watchflare system user
func CreateUser() error {
	username := "watchflare"

	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		fmt.Printf("  → User '%s' already exists\n", username)
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return createUserLinux(username)
	case "darwin":
		return createUserMacOS(username)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// createUserLinux creates a system user on Linux
func createUserLinux(username string) error {
	// Create group first
	cmd := exec.Command("groupadd", "--system", username)
	if err := cmd.Run(); err != nil {
		// Ignore if group already exists
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 9 {
			return fmt.Errorf("failed to create group: %w", err)
		}
	}

	// Create user
	cmd = exec.Command("useradd",
		"--system",
		"--gid", username,
		"--home-dir", "/var/empty",
		"--shell", "/usr/sbin/nologin",
		"--comment", "Watchflare Agent",
		username,
	)

	if err := cmd.Run(); err != nil {
		// Ignore if user already exists
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 9 {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	fmt.Printf("  → Created user '%s'\n", username)
	return nil
}

// createUserMacOS creates a system user on macOS
func createUserMacOS(username string) error {
	// Simple check: if 'id' command succeeds, user exists properly
	idCmd := exec.Command("id", "-u", username)
	if err := idCmd.Run(); err == nil {
		fmt.Printf("  → User '%s' already exists\n", username)
		return nil
	}

	// Find next available UID in system range (200-500)
	listCmd := exec.Command("dscl", ".", "-list", "/Users", "UniqueID")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Parse output to find next available UID
	usedUIDs := make(map[int]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "username    UID" (multiple spaces/tabs)
		// Split by whitespace and take the last field
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if uid, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
				if uid >= 200 && uid < 500 {
					usedUIDs[uid] = true
				}
			}
		}
	}

	// Find first available UID starting at 200
	nextUID := 200
	for usedUIDs[nextUID] {
		nextUID++
		if nextUID >= 500 {
			return fmt.Errorf("no available UID in system range (200-500)")
		}
	}

	// Create user
	commands := [][]string{
		{"dscl", ".", "-create", "/Users/" + username},
		{"dscl", ".", "-create", "/Users/" + username, "UniqueID", strconv.Itoa(nextUID)},
		{"dscl", ".", "-create", "/Users/" + username, "PrimaryGroupID", "20"}, // staff group
		{"dscl", ".", "-create", "/Users/" + username, "UserShell", "/usr/bin/false"},
		{"dscl", ".", "-create", "/Users/" + username, "RealName", "Watchflare Agent"},
		{"dscl", ".", "-create", "/Users/" + username, "NFSHomeDirectory", "/var/empty"},
		{"dscl", ".", "-create", "/Users/" + username, "Password", "*"},
	}

	// Execute all commands to create the user
	for i, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create user at step %d (cmd: %v): %w", i+1, cmdArgs, err)
		}
	}

	fmt.Printf("  → Created user '%s' (UID: %d)\n", username, nextUID)
	return nil
}

// CreateDirectories creates all necessary directories with proper permissions
func CreateDirectories() error {
	username := "watchflare"
	var groupname string

	switch runtime.GOOS {
	case "linux":
		groupname = "watchflare"
	case "darwin":
		groupname = "staff"
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Get user and group IDs
	uid, err := getUserID(username)
	if err != nil {
		return fmt.Errorf("failed to get UID for %s: %w", username, err)
	}

	gid, err := getGroupID(groupname)
	if err != nil {
		return fmt.Errorf("failed to get GID for %s: %w", groupname, err)
	}

	// Directories to create: path, owner (0=root, 1=user), permissions
	dirs := []struct {
		path  string
		owner int // 0=root, 1=user
		mode  os.FileMode
	}{
		{ConfigDir, 0, 0750},          // root:group
		{DataDir, 1, 0750},            // user:group
		{DataDir + "/wal", 1, 0750},   // user:group
	}

	// Add macOS-specific directory
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, struct {
			path  string
			owner int
			mode  os.FileMode
		}{DataDir + "/brew-cache", 1, 0750})
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir.path, err)
		}

		// Set ownership
		var ownerUID int
		if dir.owner == 0 {
			ownerUID = 0 // root
		} else {
			ownerUID = uid
		}

		if err := os.Chown(dir.path, ownerUID, gid); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", dir.path, err)
		}

		if err := os.Chmod(dir.path, dir.mode); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", dir.path, err)
		}

		fmt.Printf("  → Created %s\n", dir.path)
	}

	return nil
}

// InstallBinary copies the agent binary to the installation directory
func InstallBinary(sourcePath string) error {
	destPath := InstallDir + "/" + BinaryName

	// Skip if source and destination are the same path (binary already in place).
	// Opening a running executable for writing returns ETXTBSY on Linux.
	if sourcePath == destPath {
		fmt.Printf("  → Already installed at %s\n", destPath)
		return nil
	}

	// Open source file
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination binary: %w", err)
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Set ownership
	var group string
	switch runtime.GOOS {
	case "linux":
		group = "root"
	case "darwin":
		group = "wheel"
	}

	gid, err := getGroupID(group)
	if err != nil {
		return fmt.Errorf("failed to get GID for %s: %w", group, err)
	}

	if err := os.Chown(destPath, 0, gid); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	fmt.Printf("  → Installed to %s\n", destPath)
	return nil
}

// CreateLogFile creates the log file with proper permissions
func CreateLogFile() error {
	username := "watchflare"
	var groupname string

	switch runtime.GOOS {
	case "linux":
		groupname = "watchflare"
	case "darwin":
		groupname = "staff"
	}

	// Get user and group IDs
	uid, err := getUserID(username)
	if err != nil {
		return fmt.Errorf("failed to get UID for %s: %w", username, err)
	}

	gid, err := getGroupID(groupname)
	if err != nil {
		return fmt.Errorf("failed to get GID for %s: %w", groupname, err)
	}

	// Create or touch the log file
	file, err := os.OpenFile(LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	file.Close()

	// Set ownership
	if err := os.Chown(LogPath, uid, gid); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	if err := os.Chmod(LogPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	fmt.Printf("  → Created log file %s\n", LogPath)
	return nil
}

// RemoveFiles removes installation files
func RemoveFiles() error {
	binaryPath := InstallDir + "/" + BinaryName

	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	fmt.Printf("  → Removed %s\n", binaryPath)
	return nil
}

// RemoveDirectories removes data and config directories
func RemoveDirectories(removeData, removeConfig bool) error {
	if removeData {
		if err := os.RemoveAll(DataDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove data directory: %w", err)
		}
		fmt.Printf("  → Removed %s\n", DataDir)
	}

	if removeConfig {
		if err := os.RemoveAll(ConfigDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config directory: %w", err)
		}
		fmt.Printf("  → Removed %s\n", ConfigDir)
	}

	return nil
}

// RemoveUser removes the watchflare system user
func RemoveUser() error {
	username := "watchflare"

	switch runtime.GOOS {
	case "linux":
		// Remove user (this also removes the group if no other users use it)
		cmd := exec.Command("userdel", username)
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 6 {
				// User doesn't exist, that's fine
				return nil
			}
			return fmt.Errorf("failed to remove user: %w", err)
		}

		// Try to remove group (may fail if other users use it, that's okay)
		exec.Command("groupdel", username).Run()

	case "darwin":
		cmd := exec.Command("dscl", ".", "-delete", "/Users/"+username)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove user: %w", err)
		}
	}

	fmt.Printf("  → Removed user '%s'\n", username)
	return nil
}

// RemoveLogFile removes the agent log file
func RemoveLogFile() error {
	if err := os.Remove(LogPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove log file: %w", err)
	}

	fmt.Printf("  → Removed %s\n", LogPath)
	return nil
}

// AskConfirmation asks the user for yes/no confirmation
func AskConfirmation(prompt string) bool {
	fmt.Printf("%s (y/N): ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes" || response == "YES"
}

// GetBinaryPath returns the path to the running binary
func GetBinaryPath() (string, error) {
	// Try to get the path from /proc/self/exe (Linux) or equivalent
	if path, err := os.Readlink("/proc/self/exe"); err == nil {
		return path, nil
	}

	// Fall back to os.Executable()
	return os.Executable()
}

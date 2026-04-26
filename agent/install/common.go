package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"time"
)

const (
	execTimeout    = 10 * time.Second
	maxBinaryBytes = 100 * 1024 * 1024 // 100 MB
)

const (
	// Common paths
	InstallDir = "/usr/local/bin"
	ConfigDir  = "/etc/watchflare"
	DataDir    = "/var/lib/watchflare"
	LogPath    = "/var/log/watchflare-agent.log"
	BinaryName = "watchflare-agent"
	UserName   = "watchflare"
)

// ServiceManager defines the interface for OS-specific service management
type ServiceManager interface {
	// Install installs the systemd service
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


// CheckRoot verifies that the program is running as root
func CheckRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run as root (use sudo)")
	}
	return nil
}

// getUserID returns the UID for a username
func getUserID(username string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "id", "-u", username)
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
	username := UserName

	// Check if user already exists
	if _, err := user.Lookup(username); err == nil {
		fmt.Printf("  → User '%s' already exists\n", username)
		return nil
	}

	return createUserLinux(username)
}

// createUserLinux creates a system user on Linux
func createUserLinux(username string) error {
	// Create group first
	groupCtx, groupCancel := context.WithTimeout(context.Background(), execTimeout)
	defer groupCancel()
	cmd := exec.CommandContext(groupCtx, "groupadd", "--system", username)
	if err := cmd.Run(); err != nil {
		// Ignore if group already exists
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 9 {
			return fmt.Errorf("failed to create group: %w", err)
		}
	}

	// Create user
	userCtx, userCancel := context.WithTimeout(context.Background(), execTimeout)
	defer userCancel()
	cmd = exec.CommandContext(userCtx, "useradd",
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

// CreateDirectories creates all necessary directories with proper permissions
func CreateDirectories() error {
	uid, err := getUserID(UserName)
	if err != nil {
		return fmt.Errorf("failed to get UID for %s: %w", UserName, err)
	}

	gid, err := getGroupID(UserName)
	if err != nil {
		return fmt.Errorf("failed to get GID for %s: %w", UserName, err)
	}

	// Directories to create: path, owner (0=root, 1=user), permissions
	dirs := []struct {
		path  string
		owner int // 0=root, 1=user
		mode  os.FileMode
	}{
		{ConfigDir, 0, 0750},        // root:watchflare
		{DataDir, 1, 0750},          // watchflare:watchflare
		{DataDir + "/wal", 1, 0750}, // watchflare:watchflare
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

	// Copy file (bounded to prevent disk exhaustion from oversized source).
	// Read maxBinaryBytes+1 to detect truncation: if n > maxBinaryBytes the source is too large.
	n, copyErr := io.CopyN(dst, src, maxBinaryBytes+1)
	dst.Close()
	if copyErr != nil && copyErr != io.EOF {
		os.Remove(destPath)
		return fmt.Errorf("failed to copy binary: %w", copyErr)
	}
	if n > maxBinaryBytes {
		os.Remove(destPath)
		return fmt.Errorf("binary exceeds maximum size of %d MB", maxBinaryBytes/1024/1024)
	}

	gid, err := getGroupID("root")
	if err != nil {
		return fmt.Errorf("failed to get GID for root: %w", err)
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
	uid, err := getUserID(UserName)
	if err != nil {
		return fmt.Errorf("failed to get UID for %s: %w", UserName, err)
	}

	gid, err := getGroupID(UserName)
	if err != nil {
		return fmt.Errorf("failed to get GID for %s: %w", UserName, err)
	}

	// Create or touch the log file
	file, err := os.OpenFile(LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

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

	err := os.Remove(binaryPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}
	if err == nil {
		fmt.Printf("  → Removed %s\n", binaryPath)
	}
	return nil
}

// RemoveDirectories removes data and config directories
func RemoveDirectories(removeData, removeConfig bool) error {
	if removeData {
		err := os.RemoveAll(DataDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove data directory: %w", err)
		}
		if err == nil {
			fmt.Printf("  → Removed %s\n", DataDir)
		}
	}

	if removeConfig {
		err := os.RemoveAll(ConfigDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config directory: %w", err)
		}
		if err == nil {
			fmt.Printf("  → Removed %s\n", ConfigDir)
		}
	}

	return nil
}

// RemoveUser removes the watchflare system user
func RemoveUser() error {
	userCtx, userCancel := context.WithTimeout(context.Background(), execTimeout)
	defer userCancel()

	cmd := exec.CommandContext(userCtx, "userdel", UserName)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 6 {
			// User doesn't exist, that's fine
			return nil
		}
		return fmt.Errorf("failed to remove user: %w", err)
	}

	// Try to remove group (may fail if other users use it, that's okay)
	groupCtx, groupCancel := context.WithTimeout(context.Background(), execTimeout)
	defer groupCancel()
	_ = exec.CommandContext(groupCtx, "groupdel", UserName).Run()

	fmt.Printf("  → Removed user '%s'\n", UserName)
	return nil
}

// RemoveLogFile removes the agent log file
func RemoveLogFile() error {
	err := os.Remove(LogPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove log file: %w", err)
	}
	if err == nil {
		fmt.Printf("  → Removed %s\n", LogPath)
	}
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
	// /proc/self/exe is Linux-specific; os.Executable() is the portable fallback
	if path, err := os.Readlink("/proc/self/exe"); err == nil {
		return path, nil
	}

	// Fall back to os.Executable()
	return os.Executable()
}

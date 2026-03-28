package install

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
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

// GetServiceManager is defined in platform-specific files (common_linux.go, common_darwin.go).
// On macOS it returns an error — use Homebrew to manage the agent.

// CheckRoot verifies that the program is running as root
func CheckRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run as root (use sudo)")
	}
	return nil
}

// getUserID returns the UID for a username
func getUserID(username string) (int, error) {
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

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := dst.Close(); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
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
	cmd := exec.Command("userdel", UserName)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 6 {
			// User doesn't exist, that's fine
			return nil
		}
		return fmt.Errorf("failed to remove user: %w", err)
	}

	// Try to remove group (may fail if other users use it, that's okay)
	exec.Command("groupdel", UserName).Run()

	fmt.Printf("  → Removed user '%s'\n", UserName)
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
	// /proc/self/exe is Linux-specific; os.Executable() is the portable fallback
	if path, err := os.Readlink("/proc/self/exe"); err == nil {
		return path, nil
	}

	// Fall back to os.Executable()
	return os.Executable()
}

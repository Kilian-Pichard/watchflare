package sysinfo

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfo contains information about the system
type SystemInfo struct {
	Hostname        string
	IPv4Address     string
	IPv6Address     string
	Platform        string // "macOS", "Linux", "Windows" (user-friendly name)
	PlatformVersion string // "15.6.1", "22.04.3" (version number)
	PlatformFamily  string // "darwin", "linux", "windows" (technical family)
	Architecture    string // "arm64", "amd64", "386"
	Kernel          string // "24.6.0", "5.15.0-97-generic" (kernel version)
}

// Collect gathers system information
func Collect() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	info.Hostname = hostname

	// Get OS info
	info.PlatformFamily = runtime.GOOS
	info.Architecture = runtime.GOARCH
	info.Platform = getPlatformName()
	info.PlatformVersion = getPlatformVersion()
	info.Kernel = getKernelVersion()

	// Get IP addresses
	ipv4, ipv6, err := getIPAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get IP addresses: %w", err)
	}
	info.IPv4Address = ipv4
	info.IPv6Address = ipv6

	return info, nil
}

// getIPAddresses returns the primary IPv4 and IPv6 addresses
func getIPAddresses() (string, string, error) {
	var ipv4, ipv6 string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// IPv4
				if ipv4 == "" {
					ipv4 = ipnet.IP.String()
				}
			} else {
				// IPv6
				if ipv6 == "" && !ipnet.IP.IsLinkLocalUnicast() {
					ipv6 = ipnet.IP.String()
				}
			}
		}
	}

	return ipv4, ipv6, nil
}

// getPlatformName returns a user-friendly platform name
func getPlatformName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		return runtime.GOOS
	}
}

// getPlatformVersion returns the platform version
func getPlatformVersion() string {
	switch runtime.GOOS {
	case "darwin":
		return getMacOSVersion()
	case "linux":
		return getLinuxVersion()
	case "windows":
		return getWindowsVersion()
	default:
		return "Unknown"
	}
}

// getKernelVersion returns the kernel version
func getKernelVersion() string {
	switch runtime.GOOS {
	case "darwin":
		return getMacOSKernelVersion()
	case "linux":
		return getLinuxKernelVersion()
	case "windows":
		return getWindowsKernelVersion()
	default:
		return "Unknown"
	}
}

// macOS specific functions
func getMacOSVersion() string {
	// Use sw_vers to get macOS version
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(output))
}

func getMacOSKernelVersion() string {
	// Use uname -r to get kernel version
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(output))
}

// Linux specific functions
func getLinuxVersion() string {
	// Try to read /etc/os-release for VERSION_ID
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VERSION_ID=") {
			version := strings.TrimPrefix(line, "VERSION_ID=")
			version = strings.Trim(version, "\"")
			return version
		}
	}

	return "Unknown"
}

func getLinuxKernelVersion() string {
	// Use uname -r to get kernel version
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(output))
}

// Windows specific functions
func getWindowsVersion() string {
	// For Windows, would need to use syscalls
	return "Unknown"
}

func getWindowsKernelVersion() string {
	// For Windows, would need to use syscalls
	return "Unknown"
}

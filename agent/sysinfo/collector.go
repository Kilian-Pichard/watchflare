package sysinfo

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// SystemInfo contains information about the system
type SystemInfo struct {
	Hostname    string
	IPv4Address string
	IPv6Address string
	OS          string
	OSVersion   string
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
	info.OS = runtime.GOOS
	info.OSVersion = getOSVersion()

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

// getOSVersion returns the OS version/distribution
func getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxVersion()
	case "darwin":
		return getMacOSVersion()
	case "windows":
		return getWindowsVersion()
	default:
		return "Unknown"
	}
}

func getLinuxVersion() string {
	// Try to read /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			version := strings.TrimPrefix(line, "PRETTY_NAME=")
			version = strings.Trim(version, "\"")
			return version
		}
	}

	return "Linux"
}

func getMacOSVersion() string {
	// On macOS, we can use sw_vers but for simplicity, just return macOS
	return "macOS"
}

func getWindowsVersion() string {
	// For Windows, would need to use syscalls
	return "Windows"
}

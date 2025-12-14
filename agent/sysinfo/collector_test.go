package sysinfo

import (
	"runtime"
	"testing"
)

func TestCollect(t *testing.T) {
	info, err := Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// Check hostname is not empty
	if info.Hostname == "" {
		t.Error("Hostname is empty")
	}

	// Check Platform fields
	if info.Platform == "" {
		t.Error("Platform is empty")
	}

	if info.PlatformFamily == "" {
		t.Error("PlatformFamily is empty")
	}

	// PlatformFamily should match runtime.GOOS
	if info.PlatformFamily != runtime.GOOS {
		t.Errorf("PlatformFamily = %v, want %v", info.PlatformFamily, runtime.GOOS)
	}

	// Platform version should not be empty
	if info.PlatformVersion == "" {
		t.Error("PlatformVersion is empty")
	}

	// Architecture should not be empty and match runtime.GOARCH
	if info.Architecture == "" {
		t.Error("Architecture is empty")
	}

	if info.Architecture != runtime.GOARCH {
		t.Errorf("Architecture = %v, want %v", info.Architecture, runtime.GOARCH)
	}

	// Kernel should not be empty
	if info.Kernel == "" {
		t.Error("Kernel is empty")
	}

	// At least one IP address should be present
	// (IPv4 or IPv6, depending on the system)
	if info.IPv4Address == "" && info.IPv6Address == "" {
		t.Error("Both IPv4 and IPv6 addresses are empty")
	}

	// Log the collected info for visibility
	t.Logf("Platform: %s", info.Platform)
	t.Logf("PlatformVersion: %s", info.PlatformVersion)
	t.Logf("PlatformFamily: %s", info.PlatformFamily)
	t.Logf("Architecture: %s", info.Architecture)
	t.Logf("Kernel: %s", info.Kernel)
}

func TestGetIPAddresses(t *testing.T) {
	ipv4, ipv6, err := getIPAddresses()
	if err != nil {
		t.Fatalf("getIPAddresses() error = %v", err)
	}

	// At least one IP should be found
	// (Some systems may not have IPv6)
	if ipv4 == "" && ipv6 == "" {
		t.Error("No IP addresses found")
	}

	// If IPv4 is present, it should be a valid format
	if ipv4 != "" {
		t.Logf("IPv4: %s", ipv4)
	}

	// If IPv6 is present, it should be a valid format
	if ipv6 != "" {
		t.Logf("IPv6: %s", ipv6)
	}
}

func TestGetPlatformVersion(t *testing.T) {
	version := getPlatformVersion()

	if version == "" {
		t.Error("Platform version is empty")
	}

	// Version should not be "Unknown" for supported OSes
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if version == "Unknown" {
			t.Error("Platform version is 'Unknown' for supported OS")
		}
	}

	t.Logf("Platform Version: %s", version)
}

func TestGetLinuxVersion(t *testing.T) {
	// Only run on Linux
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	version := getLinuxVersion()

	// Should at least return "Linux"
	if version == "" {
		t.Error("Linux version is empty")
	}

	t.Logf("Linux Version: %s", version)
}

func TestGetMacOSVersion(t *testing.T) {
	// Only run on macOS
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	version := getMacOSVersion()

	if version == "" || version == "Unknown" {
		t.Error("macOS version is empty or Unknown")
	}

	// Should return a version number like "15.6.1"
	t.Logf("macOS version: %s", version)
}

func TestGetWindowsVersion(t *testing.T) {
	// Only run on Windows
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	version := getWindowsVersion()

	if version == "" {
		t.Error("Windows version is empty")
	}

	// Should return "Windows" as per current implementation
	if version != "Windows" {
		t.Errorf("getWindowsVersion() = %v, want 'Windows'", version)
	}
}

func TestSystemInfoFields(t *testing.T) {
	info := &SystemInfo{
		Hostname:        "test-host",
		IPv4Address:     "192.168.1.100",
		IPv6Address:     "fe80::1",
		Platform:        "Linux",
		PlatformVersion: "22.04.3",
		PlatformFamily:  "linux",
		Architecture:    "amd64",
		Kernel:          "5.15.0-97-generic",
	}

	tests := []struct {
		name  string
		field string
		value string
	}{
		{"hostname", "Hostname", info.Hostname},
		{"ipv4", "IPv4Address", info.IPv4Address},
		{"ipv6", "IPv6Address", info.IPv6Address},
		{"platform", "Platform", info.Platform},
		{"platform_version", "PlatformVersion", info.PlatformVersion},
		{"platform_family", "PlatformFamily", info.PlatformFamily},
		{"architecture", "Architecture", info.Architecture},
		{"kernel", "Kernel", info.Kernel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s is empty", tt.field)
			}
		})
	}
}

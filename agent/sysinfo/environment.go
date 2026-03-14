package sysinfo

import (
	"os"
	"runtime"
	"strings"
)

// EnvironmentType represents the type of environment where the agent runs
type EnvironmentType string

const (
	EnvPhysical                EnvironmentType = "physical"                  // Bare metal server
	EnvPhysicalWithContainers  EnvironmentType = "physical_with_containers"  // Physical server running containers
	EnvVM                      EnvironmentType = "vm"                        // Virtual machine
	EnvVMWithContainers        EnvironmentType = "vm_with_containers"        // VM running containers
	EnvContainer               EnvironmentType = "container"                 // Inside a container
)

// Environment holds information about the runtime environment
type Environment struct {
	Type              EnvironmentType
	IsPhysical        bool
	IsVM              bool
	IsContainer       bool
	HasDockerRunning  bool
	ContainerRuntime  string // "docker", "lxc", "podman", etc.
	Hypervisor        string // "kvm", "vmware", "virtualbox", "hyperv", "xen", etc.
}

// DetectEnvironment detects the type of environment the agent is running in
func DetectEnvironment() *Environment {
	env := &Environment{}

	// 1. Detect if running inside a container
	env.IsContainer = isRunningInContainer()
	if env.IsContainer {
		env.ContainerRuntime = detectContainerRuntime()
	}

	// 2. Detect if running in a VM
	if !env.IsContainer {
		env.IsVM = isRunningInVM()
		if env.IsVM {
			env.Hypervisor = detectHypervisor()
		}
	}

	// 3. Check if Docker is running (for VM detection)
	env.HasDockerRunning = isDockerRunning()

	// 4. Determine if physical
	env.IsPhysical = !env.IsContainer && !env.IsVM

	// 5. Determine final type
	env.Type = determineType(env)

	return env
}

// determineType determines the final environment type
func determineType(env *Environment) EnvironmentType {
	if env.IsContainer {
		return EnvContainer
	}

	if env.IsVM {
		if env.HasDockerRunning {
			return EnvVMWithContainers
		}
		return EnvVM
	}

	// Physical server
	if env.HasDockerRunning {
		return EnvPhysicalWithContainers
	}
	return EnvPhysical
}

// isRunningInContainer detects if running inside a container
func isRunningInContainer() bool {
	// Method 1: Check for /.dockerenv file (Docker)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check cgroup for container indicators
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "lxc") ||
			strings.Contains(content, "kubepods") ||
			strings.Contains(content, "podman") {
			return true
		}
	}

	// Method 3: Check if PID 1 is not init/systemd (common in containers)
	if data, err := os.ReadFile("/proc/1/cmdline"); err == nil {
		cmdline := string(data)
		// In containers, PID 1 is often not init/systemd
		if !strings.Contains(cmdline, "init") &&
			!strings.Contains(cmdline, "systemd") {
			// Additional check: look for container-specific processes
			if strings.Contains(cmdline, "bash") ||
				strings.Contains(cmdline, "sh") ||
				strings.Contains(cmdline, "container") {
				return true
			}
		}
	}

	return false
}

// detectContainerRuntime identifies the container runtime
func detectContainerRuntime() string {
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") {
			return "docker"
		}
		if strings.Contains(content, "lxc") {
			return "lxc"
		}
		if strings.Contains(content, "kubepods") {
			return "kubernetes"
		}
		if strings.Contains(content, "podman") {
			return "podman"
		}
	}

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}

	return "unknown"
}

// isRunningInVM detects if running in a virtual machine
func isRunningInVM() bool {
	// Linux: check multiple indicators
	if runtime.GOOS == "linux" {
		// Method 1: Check /sys/class/dmi/id/product_name
		if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
			product := strings.ToLower(string(data))
			if strings.Contains(product, "vmware") ||
				strings.Contains(product, "virtualbox") ||
				strings.Contains(product, "kvm") ||
				strings.Contains(product, "qemu") ||
				strings.Contains(product, "virtual") ||
				strings.Contains(product, "bochs") {
				return true
			}
		}

		// Method 2: Check /sys/class/dmi/id/sys_vendor
		if data, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
			vendor := strings.ToLower(string(data))
			if strings.Contains(vendor, "vmware") ||
				strings.Contains(vendor, "innotek") || // VirtualBox
				strings.Contains(vendor, "qemu") ||
				strings.Contains(vendor, "microsoft") || // Hyper-V
				strings.Contains(vendor, "xen") {
				return true
			}
		}

		// Method 3: Check systemd-detect-virt if available
		// (We'll add this if needed)
	}

	// macOS: Check for virtualization
	if runtime.GOOS == "darwin" {
		// macOS VMs are less common, but we can check sysctl
		// This would require additional implementation
	}

	return false
}

// detectHypervisor identifies the hypervisor type
func detectHypervisor() string {
	if runtime.GOOS == "linux" {
		// Check product name
		if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
			product := strings.ToLower(string(data))
			if strings.Contains(product, "vmware") {
				return "vmware"
			}
			if strings.Contains(product, "virtualbox") {
				return "virtualbox"
			}
			if strings.Contains(product, "kvm") || strings.Contains(product, "qemu") {
				return "kvm"
			}
		}

		// Check sys vendor
		if data, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
			vendor := strings.ToLower(string(data))
			if strings.Contains(vendor, "vmware") {
				return "vmware"
			}
			if strings.Contains(vendor, "innotek") {
				return "virtualbox"
			}
			if strings.Contains(vendor, "qemu") {
				return "kvm"
			}
			if strings.Contains(vendor, "microsoft") {
				return "hyperv"
			}
			if strings.Contains(vendor, "xen") {
				return "xen"
			}
		}
	}

	return "unknown"
}

// isDockerRunning checks if Docker daemon is running on the system
func isDockerRunning() bool {
	// Check if docker.sock exists
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return true
	}

	// Check if docker command exists and works
	// (We could execute `docker ps` but that's expensive)
	// For now, just check the socket
	return false
}

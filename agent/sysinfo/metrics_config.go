package sysinfo

// MetricsConfig defines which metrics to collect based on environment type
type MetricsConfig struct {
	CollectCPU         bool
	CollectMemory      bool
	CollectDisk        bool
	CollectDiskIO      bool
	CollectNetwork     bool
	CollectSwap        bool
	CollectLoadAvg     bool
	CollectTemperature bool

	// Container-specific
	CollectContainerCPU     bool
	CollectContainerMemory  bool
	CollectContainerNetwork bool

	// Docker-specific (for VMs running Docker)
	CollectDockerCPU     bool
	CollectDockerMemory  bool
	CollectDockerNetwork bool
}

// GetMetricsConfig returns the appropriate metrics configuration based on environment
func GetMetricsConfig(env *Environment) *MetricsConfig {
	config := &MetricsConfig{}

	switch env.Type {
	case EnvPhysical:
		// Physical server: collect everything
		config.CollectCPU = true
		config.CollectMemory = true
		config.CollectDisk = true
		config.CollectDiskIO = true
		config.CollectNetwork = true
		config.CollectSwap = true
		config.CollectLoadAvg = true
		config.CollectTemperature = true

	case EnvPhysicalWithContainers:
		// Physical server running Docker: collect everything + Docker metrics
		config.CollectCPU = true
		config.CollectMemory = true
		config.CollectDisk = true
		config.CollectDiskIO = true
		config.CollectNetwork = true
		config.CollectSwap = true
		config.CollectLoadAvg = true
		config.CollectTemperature = true

		// Docker-specific metrics
		config.CollectDockerCPU = true
		config.CollectDockerMemory = true
		config.CollectDockerNetwork = true

	case EnvVM:
		// VM without containers: collect most things except temperature
		config.CollectCPU = true
		config.CollectMemory = true
		config.CollectDisk = true
		config.CollectDiskIO = true
		config.CollectNetwork = true
		config.CollectLoadAvg = true
		config.CollectSwap = false       // VMs often don't use swap
		config.CollectTemperature = false // Can't read physical sensors

	case EnvVMWithContainers:
		// VM running Docker: collect VM metrics + Docker metrics
		config.CollectCPU = true
		config.CollectMemory = true
		config.CollectDisk = true
		config.CollectDiskIO = true
		config.CollectNetwork = true
		config.CollectLoadAvg = true
		config.CollectSwap = false
		config.CollectTemperature = false

		// Docker-specific metrics
		config.CollectDockerCPU = true
		config.CollectDockerMemory = true
		config.CollectDockerNetwork = true

	case EnvContainer:
		// Container: collect limited metrics
		// Only what's relevant to the container itself
		config.CollectCPU = true // Container CPU usage
		config.CollectMemory = true // Container memory usage
		config.CollectDisk = false // Disk is shared with host - don't report
		config.CollectDiskIO = false // I/O is shared
		config.CollectNetwork = false // Network is complex in containers
		config.CollectSwap = false
		config.CollectLoadAvg = true // Load avg might be relevant
		config.CollectTemperature = false

		// Mark as container metrics
		config.CollectContainerCPU = true
		config.CollectContainerMemory = true
	}

	return config
}

// String returns a human-readable description of the environment
func (e *Environment) String() string {
	switch e.Type {
	case EnvPhysical:
		return "Physical Server"
	case EnvPhysicalWithContainers:
		return "Physical Server with Docker"
	case EnvVM:
		return "Virtual Machine (" + e.Hypervisor + ")"
	case EnvVMWithContainers:
		return "Virtual Machine with Docker (" + e.Hypervisor + ")"
	case EnvContainer:
		return "Container (" + e.ContainerRuntime + ")"
	default:
		return "Unknown"
	}
}

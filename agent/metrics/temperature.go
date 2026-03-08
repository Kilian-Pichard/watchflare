package metrics

import (
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

// getCPUTemperature returns the CPU temperature in Celsius, or 0 if unavailable
func getCPUTemperature() (float64, error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0, err
	}

	// Look for CPU temperature sensors (in priority order)
	for _, temp := range temps {
		if isCPUSensor(temp.SensorKey) && temp.Temperature > 0 {
			return temp.Temperature, nil
		}
	}

	return 0, nil
}

// isCPUSensor checks if a sensor key corresponds to a CPU temperature sensor
func isCPUSensor(sensorKey string) bool {
	lower := strings.ToLower(sensorKey)
	cpuPatterns := []string{
		"coretemp",      // Intel
		"k10temp",       // AMD
		"cpu_thermal",   // ARM / Raspberry Pi
		"package id 0",  // Intel package temp
		"tctl",          // AMD Ryzen (Tctl)
		"cpu temp",      // Generic fallback (exact "cpu temp" to avoid matching "cpu_fan" etc.)
		"cpu die",       // Some systems
	}

	for _, pattern := range cpuPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

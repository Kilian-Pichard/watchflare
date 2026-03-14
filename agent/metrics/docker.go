package metrics

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ContainerMetric represents metrics for a single Docker container
type ContainerMetric struct {
	ContainerID          string
	ContainerName        string
	Image                string
	CPUPercent           float64
	MemoryUsedBytes      uint64
	MemoryLimitBytes     uint64
	NetworkRxBytesPerSec uint64
	NetworkTxBytesPerSec uint64
}

// dockerStatsResponse matches the Docker API /containers/{id}/stats response
type dockerStatsResponse struct {
	CPUStats    dockerCPUStats    `json:"cpu_stats"`
	PreCPUStats dockerCPUStats    `json:"precpu_stats"`
	MemoryStats dockerMemoryStats `json:"memory_stats"`
	Networks    map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

type dockerCPUStats struct {
	CPUUsage struct {
		TotalUsage uint64 `json:"total_usage"`
	} `json:"cpu_usage"`
	SystemCPUUsage uint64 `json:"system_cpu_usage"`
	OnlineCPUs     uint64 `json:"online_cpus"`
}

type dockerMemoryStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
	Stats struct {
		InactiveFile uint64 `json:"inactive_file"`
		Cache        uint64 `json:"cache"`
	} `json:"stats"`
}

type dockerContainer struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
	Image string   `json:"Image"`
	State string   `json:"State"`
}

// dockerHTTPClient creates an HTTP client that connects via Unix socket
func dockerHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
		Timeout: 10 * time.Second,
	}
}

// CollectContainerMetrics collects metrics for all running Docker containers
func CollectContainerMetrics(tracker *DeltaTracker) ([]ContainerMetric, error) {
	httpClient := dockerHTTPClient()

	// List running containers
	resp, err := httpClient.Get("http://localhost/v1.43/containers/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, nil
	}

	var result []ContainerMetric

	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		stats, err := getContainerStats(httpClient, c.ID)
		if err != nil {
			log.Printf("Warning: Failed to get stats for container %s: %v", c.ID[:12], err)
			continue
		}

		// Compute CPU percentage
		cpuPercent := computeCPUPercent(stats)

		// Compute memory (exclude cache for actual usage)
		memUsed := stats.MemoryStats.Usage
		if stats.MemoryStats.Stats.InactiveFile > 0 {
			memUsed -= stats.MemoryStats.Stats.InactiveFile
		}

		// Sum network counters across all interfaces
		var totalRx, totalTx uint64
		for _, netStats := range stats.Networks {
			totalRx += netStats.RxBytes
			totalTx += netStats.TxBytes
		}

		// Compute network rate using delta tracker
		now := time.Now()
		rxRate, txRate := tracker.ComputeContainerNetworkRate(c.ID, totalRx, totalTx, now)

		// Clean container name (remove leading /)
		name := c.ID[:12]
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		result = append(result, ContainerMetric{
			ContainerID:          c.ID[:12],
			ContainerName:        name,
			Image:                c.Image,
			CPUPercent:           cpuPercent,
			MemoryUsedBytes:      memUsed,
			MemoryLimitBytes:     stats.MemoryStats.Limit,
			NetworkRxBytesPerSec: rxRate,
			NetworkTxBytesPerSec: txRate,
		})
	}

	return result, nil
}

// getContainerStats fetches one-shot stats for a container
func getContainerStats(client *http.Client, containerID string) (*dockerStatsResponse, error) {
	resp, err := client.Get("http://localhost/v1.43/containers/" + containerID + "/stats?stream=false")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stats dockerStatsResponse
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// computeCPUPercent calculates CPU usage percentage from Docker stats
func computeCPUPercent(stats *dockerStatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	if systemDelta <= 0 || cpuDelta <= 0 {
		return 0
	}

	numCPUs := stats.CPUStats.OnlineCPUs
	if numCPUs == 0 {
		numCPUs = 1
	}

	return (cpuDelta / systemDelta) * float64(numCPUs) * 100.0
}

package docker

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// dockerStats represents the raw stats response from Docker API
type dockerStats struct {
	Read     time.Time `json:"read"`
	PreRead  time.Time `json:"preread"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// StreamContainerStats streams container stats through a channel
func (c *MultiHostClient) StreamContainerStats(ctx context.Context, hostName, containerID string) (<-chan models.ContainerStats, <-chan error) {
	statsCh := make(chan models.ContainerStats)
	errCh := make(chan error, 1)

	go func() {
		defer close(statsCh)
		defer close(errCh)

		apiClient, err := c.GetClient(hostName)
		if err != nil {
			errCh <- err
			return
		}

		stats, err := apiClient.ContainerStats(ctx, containerID, true)
		if err != nil {
			errCh <- err
			return
		}
		defer stats.Body.Close()

		decoder := json.NewDecoder(stats.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var raw dockerStats
				if err := decoder.Decode(&raw); err != nil {
					if err == io.EOF {
						return
					}
					errCh <- err
					return
				}

				parsed := parseDockerStats(raw, containerID, hostName)
				select {
				case statsCh <- parsed:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return statsCh, errCh
}

// GetContainerStatsOnce returns a single stats snapshot (non-streaming)
func (c *MultiHostClient) GetContainerStatsOnce(ctx context.Context, hostName, containerID string) (*models.ContainerStats, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	stats, err := apiClient.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var raw dockerStats
	if err := json.NewDecoder(stats.Body).Decode(&raw); err != nil {
		return nil, err
	}

	parsed := parseDockerStats(raw, containerID, hostName)
	return &parsed, nil
}

// GetAllContainersStats returns stats for all running containers on a host
func (c *MultiHostClient) GetAllContainersStats(ctx context.Context, hostName string) ([]models.ContainerStats, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	containers, err := apiClient.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var allStats []models.ContainerStats
	for _, ctr := range containers {
		if ctr.State != "running" {
			continue
		}

		stats, err := c.GetContainerStatsOnce(ctx, hostName, ctr.ID)
		if err != nil {
			continue // Skip containers we can't get stats for
		}
		allStats = append(allStats, *stats)
	}

	return allStats, nil
}

// parseDockerStats converts raw Docker stats to our model
func parseDockerStats(raw dockerStats, containerID, host string) models.ContainerStats {
	// Calculate CPU percentage
	cpuPercent := calculateCPUPercent(raw)

	// Calculate memory percentage
	var memPercent float64
	if raw.MemoryStats.Limit > 0 {
		memPercent = float64(raw.MemoryStats.Usage) / float64(raw.MemoryStats.Limit) * 100
	}

	// Aggregate network stats across all interfaces
	var netRx, netTx uint64
	for _, net := range raw.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	// Aggregate block I/O stats
	var blockRead, blockWrite uint64
	for _, bio := range raw.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "Read", "read":
			blockRead += bio.Value
		case "Write", "write":
			blockWrite += bio.Value
		}
	}

	return models.ContainerStats{
		ContainerID:   containerID,
		Host:          host,
		CPUPercent:    cpuPercent,
		MemoryUsage:   raw.MemoryStats.Usage,
		MemoryLimit:   raw.MemoryStats.Limit,
		MemoryPercent: memPercent,
		NetworkRx:     netRx,
		NetworkTx:     netTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          raw.PidsStats.Current,
		Timestamp:     raw.Read.Unix(),
	}
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats
func calculateCPUPercent(raw dockerStats) float64 {
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(raw.CPUStats.OnlineCPUs) * 100
		return cpuPercent
	}
	return 0
}

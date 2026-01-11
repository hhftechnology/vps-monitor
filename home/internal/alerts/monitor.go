package alerts

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// Monitor handles background monitoring and alerting
type Monitor struct {
	docker  *docker.MultiHostClient
	config  *config.AlertConfig
	history *AlertHistory
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Track container states for detecting changes
	containerStates map[string]string // key: host:containerID, value: state
	statesMu        sync.RWMutex
}

// NewMonitor creates a new alert monitor
func NewMonitor(dockerClient *docker.MultiHostClient, alertConfig *config.AlertConfig) *Monitor {
	return &Monitor{
		docker:          dockerClient,
		config:          alertConfig,
		history:         NewAlertHistory(100), // Keep last 100 alerts
		stopCh:          make(chan struct{}),
		containerStates: make(map[string]string),
	}
}

// Start begins the background monitoring
func (m *Monitor) Start() {
	if !m.config.Enabled {
		log.Println("Alert monitoring is disabled")
		return
	}

	log.Printf("Starting alert monitor (interval: %s, CPU threshold: %.1f%%, Memory threshold: %.1f%%)",
		m.config.CheckInterval, m.config.CPUThreshold, m.config.MemoryThreshold)

	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop gracefully stops the monitor
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	log.Println("Alert monitor stopped")
}

// GetHistory returns the alert history
func (m *Monitor) GetHistory() *AlertHistory {
	return m.history
}

// monitorLoop is the main monitoring loop
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	// Initial check
	m.checkAll()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopCh:
			return
		}
	}
}

// checkAll performs all monitoring checks
func (m *Monitor) checkAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m.checkContainerStates(ctx)
	m.checkResourceThresholds(ctx)
}

// checkContainerStates checks for container state changes
func (m *Monitor) checkContainerStates(ctx context.Context) {
	containersMap, _, err := m.docker.ListContainersAllHosts(ctx)
	if err != nil {
		log.Printf("Alert monitor: failed to list containers: %v", err)
		return
	}

	m.statesMu.Lock()
	defer m.statesMu.Unlock()

	// Track current containers
	currentContainers := make(map[string]struct{})

	for hostName, containers := range containersMap {
		for _, ctr := range containers {
			key := fmt.Sprintf("%s:%s", hostName, ctr.ID)
			currentContainers[key] = struct{}{}

			containerName := ctr.ID[:12]
			if len(ctr.Names) > 0 {
				containerName = strings.TrimPrefix(ctr.Names[0], "/")
			}

			// Check if state changed
			if prevState, exists := m.containerStates[key]; exists {
				if prevState != ctr.State {
					// State changed
					if ctr.State == "exited" || ctr.State == "dead" {
						m.triggerAlert(models.Alert{
							ID:            uuid.New().String(),
							Type:          models.AlertContainerStopped,
							ContainerID:   ctr.ID,
							ContainerName: containerName,
							Host:          hostName,
							Message:       fmt.Sprintf("Container %s stopped (was: %s, now: %s)", containerName, prevState, ctr.State),
							Timestamp:     time.Now().Unix(),
						})
					} else if ctr.State == "running" && (prevState == "exited" || prevState == "dead" || prevState == "created") {
						m.triggerAlert(models.Alert{
							ID:            uuid.New().String(),
							Type:          models.AlertContainerStarted,
							ContainerID:   ctr.ID,
							ContainerName: containerName,
							Host:          hostName,
							Message:       fmt.Sprintf("Container %s started", containerName),
							Timestamp:     time.Now().Unix(),
						})
					}
				}
			}

			m.containerStates[key] = ctr.State
		}
	}

	// Clean up containers that no longer exist
	for key := range m.containerStates {
		if _, exists := currentContainers[key]; !exists {
			delete(m.containerStates, key)
		}
	}
}

// checkResourceThresholds checks CPU and memory thresholds
func (m *Monitor) checkResourceThresholds(ctx context.Context) {
	containersMap, _, err := m.docker.ListContainersAllHosts(ctx)
	if err != nil {
		return
	}

	for hostName, containers := range containersMap {
		for _, ctr := range containers {
			if ctr.State != "running" {
				continue
			}

			stats, err := m.docker.GetContainerStatsOnce(ctx, hostName, ctr.ID)
			if err != nil {
				continue
			}

			containerName := ctr.ID[:12]
			if len(ctr.Names) > 0 {
				containerName = strings.TrimPrefix(ctr.Names[0], "/")
			}

			// Check CPU threshold
			if stats.CPUPercent > m.config.CPUThreshold {
				m.triggerAlert(models.Alert{
					ID:            uuid.New().String(),
					Type:          models.AlertCPUThreshold,
					ContainerID:   ctr.ID,
					ContainerName: containerName,
					Host:          hostName,
					Message:       fmt.Sprintf("Container %s CPU usage (%.1f%%) exceeds threshold (%.1f%%)", containerName, stats.CPUPercent, m.config.CPUThreshold),
					Value:         stats.CPUPercent,
					Threshold:     m.config.CPUThreshold,
					Timestamp:     time.Now().Unix(),
				})
			}

			// Check memory threshold
			if stats.MemoryPercent > m.config.MemoryThreshold {
				m.triggerAlert(models.Alert{
					ID:            uuid.New().String(),
					Type:          models.AlertMemoryThreshold,
					ContainerID:   ctr.ID,
					ContainerName: containerName,
					Host:          hostName,
					Message:       fmt.Sprintf("Container %s memory usage (%.1f%%) exceeds threshold (%.1f%%)", containerName, stats.MemoryPercent, m.config.MemoryThreshold),
					Value:         stats.MemoryPercent,
					Threshold:     m.config.MemoryThreshold,
					Timestamp:     time.Now().Unix(),
				})
			}
		}
	}
}

// triggerAlert handles a new alert
func (m *Monitor) triggerAlert(alert models.Alert) {
	log.Printf("Alert: %s - %s", alert.Type, alert.Message)

	// Add to history
	m.history.Add(alert)

	// Send webhook
	if m.config.WebhookURL != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := SendWebhook(ctx, m.config.WebhookURL, alert); err != nil {
				log.Printf("Failed to send webhook for alert %s: %v", alert.ID, err)
			}
		}()
	}
}

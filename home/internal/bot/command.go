package bot

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

type commandHandler struct {
	registry *services.Registry
}

func newCommandHandler(registry *services.Registry) *commandHandler {
	return &commandHandler{registry: registry}
}

func (h *commandHandler) handle(text string) string {
	switch {
	case strings.HasPrefix(text, "/help"), strings.HasPrefix(text, "/start"):
		return "Available commands:\n/status - current container health with history\n/critical - latest critical alerts\n/help - command list"
	case strings.HasPrefix(text, "/critical"):
		return h.buildCriticalMessage()
	case strings.HasPrefix(text, "/status"):
		return h.buildStatusMessage()
	default:
		return "Unknown command. Use /help."
	}
}

func (h *commandHandler) buildCriticalMessage() string {
	if h.registry == nil {
		return "Alert monitoring is disabled."
	}

	monitor := h.registry.Alerts()
	if monitor == nil {
		return "Alert monitoring is disabled."
	}

	alertsList := monitor.GetHistory().GetAll()
	critical := make([]models.Alert, 0, len(alertsList))
	for _, alert := range alertsList {
		if alert.Type == models.AlertCPUThreshold || alert.Type == models.AlertMemoryThreshold {
			critical = append(critical, alert)
		}
	}

	if len(critical) == 0 {
		return "No critical alerts."
	}

	sort.SliceStable(critical, func(i, j int) bool {
		return critical[i].Timestamp > critical[j].Timestamp
	})

	var lines []string
	lines = append(lines, "Critical alerts:")
	for _, alert := range critical[:min(5, len(critical))] {
		lines = append(lines, fmt.Sprintf("- %s on %s (%s)", alert.ContainerName, alert.Host, alert.Type))
	}
	return strings.Join(lines, "\n")
}

func (h *commandHandler) buildStatusMessage() string {
	if h.registry == nil {
		return "Docker client unavailable."
	}

	dockerClient, release := h.registry.AcquireDocker()
	defer release()
	if dockerClient == nil {
		return "Docker client unavailable."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containersMap, _, err := dockerClient.ListContainersAllHosts(ctx)
	if err != nil {
		return fmt.Sprintf("Failed to list containers: %v", err)
	}

	type containerLine struct {
		name string
		cpu  float64
		line string
	}

	var lines []containerLine
	total := 0
	running := 0
	history := h.registry.Alerts()
	var historyManager interface {
		Get1hAverages(string, string) (float64, float64, bool)
		Get12hAverages(string, string) (float64, float64, bool)
	}
	if history != nil {
		historyManager = history.GetStatsHistory()
	}

	for hostName, containers := range containersMap {
		for _, ctr := range containers {
			total++
			if ctr.State != "running" {
				continue
			}
			running++

			stats, err := dockerClient.GetContainerStatsOnce(ctx, hostName, ctr.ID)
			if err != nil {
				continue
			}

			name := ctr.ID[:12]
			if len(ctr.Names) > 0 {
				name = strings.TrimPrefix(ctr.Names[0], "/")
			}

			line := fmt.Sprintf("- %s@%s CPU %.1f%% MEM %.1f%%", name, hostName, stats.CPUPercent, stats.MemoryPercent)
			if historyManager != nil {
				cpu1h, mem1h, has1h := historyManager.Get1hAverages(hostName, ctr.ID)
				cpu12h, mem12h, has12h := historyManager.Get12hAverages(hostName, ctr.ID)
				line = appendHistoryAverages(line, cpu1h, mem1h, has1h, cpu12h, mem12h, has12h)
			}

			lines = append(lines, containerLine{name: name, cpu: stats.CPUPercent, line: line})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].cpu > lines[j].cpu
	})

	message := []string{
		fmt.Sprintf("Containers: %d total, %d running", total, running),
	}
	for _, line := range lines[:min(5, len(lines))] {
		message = append(message, line.line)
	}
	return strings.Join(message, "\n")
}

func appendHistoryAverages(line string, cpu1h, mem1h float64, has1h bool, cpu12h, mem12h float64, has12h bool) string {
	if has1h {
		line += fmt.Sprintf(" | 1h %.1f/%.1f", cpu1h, mem1h)
	}
	if has12h {
		line += fmt.Sprintf(" | 12h %.1f/%.1f", cpu12h, mem12h)
	}
	return line
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

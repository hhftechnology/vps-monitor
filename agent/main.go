// agent/main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Metrics holds all the system metrics we want to collect.
type Metrics struct {
	AgentID       string                 `json:"agent_id"`
	Hostname      string                 `json:"hostname"`
	Uptime        uint64                 `json:"uptime"`
	CPUUsage      float64                `json:"cpu_usage"`
	Memory        *mem.VirtualMemoryStat `json:"memory"`
	Disk          *disk.UsageStat        `json:"disk"`
	Network       []net.IOCountersStat   `json:"network"`
	Processes     []*ProcessInfo         `json:"processes"`
	AgentInfo     *AgentInfo             `json:"agent_info"`
}

// ProcessInfo holds simplified information about a running process.
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
}

// AgentInfo holds information about the agent itself
type AgentInfo struct {
	Version       string            `json:"version"`
	GoVersion     string            `json:"go_version"`
	NumGoroutines int               `json:"num_goroutines"`
	MemStats      map[string]uint64 `json:"mem_stats"`
	StartTime     time.Time         `json:"start_time"`
}

const (
	// Agent version
	AgentVersion = "1.1.0"
	
	// Default intervals
	DefaultMetricsInterval = 10 * time.Second
	DefaultRetryInterval   = 5 * time.Second
	MaxRetryAttempts      = 5
	
	// HTTP client settings
	HTTPTimeout = 30 * time.Second
)

type Agent struct {
	agentID       string
	homeServerURL string
	client        *http.Client
	hostname      string
	retryCount    int
	startTime     time.Time
}

func NewAgent(homeServerURL string) *Agent {
	hostname, _ := os.Hostname()
	
	// Get agent ID from environment variable or use hostname as fallback
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = hostname
	}
	
	// Add a suffix if AGENT_NAME is provided
	if agentName := os.Getenv("AGENT_NAME"); agentName != "" {
		agentID = agentName
	}
	
	return &Agent{
		agentID:       agentID,
		homeServerURL: homeServerURL,
		client: &http.Client{
			Timeout: HTTPTimeout,
		},
		hostname:  hostname,
		startTime: time.Now(),
	}
}

func main() {
	homeServerURL := os.Getenv("HOME_SERVER_URL")
	if homeServerURL == "" {
		log.Fatal("HOME_SERVER_URL environment variable is not set.")
	}

	agent := NewAgent(homeServerURL)
	log.Printf("Starting VPS Monitor Agent v%s", AgentVersion)
	log.Printf("Agent ID: %s", agent.agentID)
	log.Printf("Hostname: %s", agent.hostname)
	log.Printf("Will report to: %s", homeServerURL)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start metrics collection
	go agent.collectAndSendMetrics(ctx)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping agent...")
	cancel()

	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)
	log.Printf("Agent %s stopped", agent.agentID)
}

func (a *Agent) collectAndSendMetrics(ctx context.Context) {
	ticker := time.NewTicker(DefaultMetricsInterval)
	defer ticker.Stop()

	// Send initial metrics immediately
	a.sendMetricsWithRetry(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendMetricsWithRetry(ctx)
		}
	}
}

func (a *Agent) sendMetricsWithRetry(ctx context.Context) {
	for attempt := 1; attempt <= MaxRetryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		metrics, err := a.collectMetrics()
		if err != nil {
			log.Printf("Error collecting metrics (attempt %d/%d): %v", attempt, MaxRetryAttempts, err)
			if attempt < MaxRetryAttempts {
				time.Sleep(DefaultRetryInterval)
				continue
			}
			return
		}

		if err := a.sendMetrics(metrics); err != nil {
			log.Printf("Error sending metrics (attempt %d/%d): %v", attempt, MaxRetryAttempts, err)
			if attempt < MaxRetryAttempts {
				time.Sleep(DefaultRetryInterval)
				continue
			}
			a.retryCount++
			return
		}

		// Success - reset retry count
		if a.retryCount > 0 {
			log.Printf("Successfully reconnected after %d failed attempts", a.retryCount)
			a.retryCount = 0
		}
		return
	}
}

// collectMetrics gathers all the system data.
func (a *Agent) collectMetrics() (*Metrics, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	network, err := net.IOCounters(true)
	if err != nil {
		log.Printf("Warning: failed to get network info: %v", err)
		network = []net.IOCountersStat{} // Continue without network data
	}

	processes, err := a.getProcessList()
	if err != nil {
		log.Printf("Warning: failed to get process list: %v", err)
		processes = []*ProcessInfo{} // Continue without process data
	}

	agentInfo := a.getAgentInfo()

	return &Metrics{
		AgentID:   a.agentID,
		Hostname:  a.hostname,
		Uptime:    hostInfo.Uptime,
		CPUUsage:  cpuUsage[0],
		Memory:    memory,
		Disk:      diskUsage,
		Network:   network,
		Processes: processes,
		AgentInfo: agentInfo,
	}, nil
}

// getProcessList gets a list of running processes.
func (a *Agent) getProcessList() ([]*ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []*ProcessInfo
	processCount := 0
	maxProcesses := 30 // Limit to top 30 processes

	for _, pid := range pids {
		if processCount >= maxProcesses {
			break
		}

		p, err := process.NewProcess(pid)
		if err != nil {
			continue // Process might have terminated
		}

		name, err := p.Name()
		if err != nil {
			continue
		}

		cpuPercent, err := p.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		memPercent, err := p.MemoryPercent()
		if err != nil {
			memPercent = 0
		}

		processes = append(processes, &ProcessInfo{
			PID:           pid,
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryPercent: memPercent,
		})
		processCount++
	}

	return processes, nil
}

// getAgentInfo gets information about the agent itself
func (a *Agent) getAgentInfo() *AgentInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Create a simplified version of memory stats
	simplifiedMemStats := map[string]uint64{
		"alloc":       memStats.Alloc,
		"total_alloc": memStats.TotalAlloc,
		"sys":         memStats.Sys,
		"num_gc":      uint64(memStats.NumGC),
	}

	return &AgentInfo{
		Version:       AgentVersion,
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemStats:      simplifiedMemStats,
		StartTime:     a.startTime,
	}
}

// sendMetrics sends the collected data to the home server.
func (a *Agent) sendMetrics(metrics *Metrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequest("POST", a.homeServerURL+"/api/metrics", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("VPS-Monitor-Agent/%s", AgentVersion))
	req.Header.Set("X-Agent-ID", a.agentID)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	log.Printf("[%s] Successfully sent metrics (CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%)", 
		a.agentID, metrics.CPUUsage, metrics.Memory.UsedPercent, metrics.Disk.UsedPercent)
	return nil
}
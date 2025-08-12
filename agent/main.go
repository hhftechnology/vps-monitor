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
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	DockerStats   []*DockerContainerStat `json:"docker_stats"`
	AgentInfo     *AgentInfo             `json:"agent_info"`
	SystemInfo    *SystemInfo            `json:"system_info"`
}

// ProcessInfo holds simplified information about a running process.
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	Command       string  `json:"command"`
}

// DockerContainerStat holds Docker container statistics
type DockerContainerStat struct {
	ContainerID   string  `json:"container_id"`
	Name          string  `json:"name"`
	CPUPercent    string  `json:"cpu_percent"`
	MemoryUsage   string  `json:"memory_usage"`
	MemoryLimit   string  `json:"memory_limit"`
	MemoryPercent string  `json:"memory_percent"`
	NetworkIO     string  `json:"network_io"`
	BlockIO       string  `json:"block_io"`
	PIDs          string  `json:"pids"`
}

// SystemInfo holds additional system information
type SystemInfo struct {
	TotalProcesses   int    `json:"total_processes"`
	DockerAvailable  bool   `json:"docker_available"`
	KernelVersion    string `json:"kernel_version"`
	OSRelease        string `json:"os_release"`
	Architecture     string `json:"architecture"`
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

	dockerStats := a.getDockerStats()
	systemInfo := a.getSystemInfo()
	agentInfo := a.getAgentInfo()

	return &Metrics{
		AgentID:     a.agentID,
		Hostname:    a.hostname,
		Uptime:      hostInfo.Uptime,
		CPUUsage:    cpuUsage[0],
		Memory:      memory,
		Disk:        diskUsage,
		Network:     network,
		Processes:   processes,
		DockerStats: dockerStats,
		SystemInfo:  systemInfo,
		AgentInfo:   agentInfo,
	}, nil
}

// getProcessList gets a list of running processes using multiple methods
func (a *Agent) getProcessList() ([]*ProcessInfo, error) {
	var processes []*ProcessInfo
	
	// Method 1: Try using ps command for more comprehensive process list
	if psProcesses := a.getProcessesFromPS(); len(psProcesses) > 0 {
		processes = psProcesses
	} else {
		// Method 2: Fallback to gopsutil
		psutilProcesses, err := a.getProcessesFromPsutil()
		if err != nil {
			return nil, err
		}
		processes = psutilProcesses
	}

	// Sort by CPU usage (descending)
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	// Limit to top 30 processes
	if len(processes) > 30 {
		processes = processes[:30]
	}

	return processes, nil
}

// getProcessesFromPS gets processes using ps command
func (a *Agent) getProcessesFromPS() []*ProcessInfo {
	cmd := exec.Command("ps", "aux", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to run ps command: %v", err)
		return nil
	}

	lines := strings.Split(string(output), "\n")
	var processes []*ProcessInfo

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		// Parse PID
		pid, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			continue
		}

		// Parse CPU percentage
		cpuPercent, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			cpuPercent = 0
		}

		// Parse memory percentage
		memPercent, err := strconv.ParseFloat(fields[3], 32)
		if err != nil {
			memPercent = 0
		}

		// Process name and command
		command := strings.Join(fields[10:], " ")
		name := fields[10]
		if len(name) > 50 {
			name = name[:50] + "..."
		}

		processes = append(processes, &ProcessInfo{
			PID:           int32(pid),
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryPercent: float32(memPercent),
			Command:       command,
		})
	}

	return processes
}

// getProcessesFromPsutil gets processes using gopsutil (fallback)
func (a *Agent) getProcessesFromPsutil() ([]*ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []*ProcessInfo
	processCount := 0
	maxProcesses := 50

	for _, pid := range pids {
		if processCount >= maxProcesses {
			break
		}

		p, err := process.NewProcess(pid)
		if err != nil {
			continue
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

		memInfo, _ := p.MemoryInfo()
		memoryMB := float64(0)
		if memInfo != nil {
			memoryMB = float64(memInfo.RSS) / 1024 / 1024
		}

		cmdline, _ := p.Cmdline()

		processes = append(processes, &ProcessInfo{
			PID:           pid,
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryPercent: memPercent,
			MemoryMB:      memoryMB,
			Command:       cmdline,
		})
		processCount++
	}

	return processes, nil
}

// getDockerStats gets Docker container statistics
func (a *Agent) getDockerStats() []*DockerContainerStat {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", 
		"table {{.Container}}\t{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}")
	
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Docker not available or failed to get stats: %v", err)
		return []*DockerContainerStat{}
	}

	lines := strings.Split(string(output), "\n")
	var dockerStats []*DockerContainerStat

	// Skip header line
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// Parse memory usage
		memUsage := strings.Split(fields[3], "/")
		memoryUsage := fields[3]
		memoryLimit := ""
		if len(memUsage) == 2 {
			memoryUsage = strings.TrimSpace(memUsage[0])
			memoryLimit = strings.TrimSpace(memUsage[1])
		}

		dockerStats = append(dockerStats, &DockerContainerStat{
			ContainerID:   fields[0],
			Name:          fields[1],
			CPUPercent:    fields[2],
			MemoryUsage:   memoryUsage,
			MemoryLimit:   memoryLimit,
			MemoryPercent: fields[4],
			NetworkIO:     fields[5],
			BlockIO:       fields[6],
			PIDs:          fields[7],
		})
	}

	return dockerStats
}

// getSystemInfo gets additional system information
func (a *Agent) getSystemInfo() *SystemInfo {
	// Get total process count
	totalProcesses := 0
	if cmd := exec.Command("ps", "-e", "--no-headers"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			totalProcesses = len(strings.Split(strings.TrimSpace(string(output)), "\n"))
		}
	}

	// Check if Docker is available
	dockerAvailable := false
	if cmd := exec.Command("docker", "version"); cmd != nil {
		if err := cmd.Run(); err == nil {
			dockerAvailable = true
		}
	}

	// Get kernel version
	kernelVersion := ""
	if cmd := exec.Command("uname", "-r"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			kernelVersion = strings.TrimSpace(string(output))
		}
	}

	// Get OS release
	osRelease := ""
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osRelease = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				break
			}
		}
	}

	return &SystemInfo{
		TotalProcesses:  totalProcesses,
		DockerAvailable: dockerAvailable,
		KernelVersion:   kernelVersion,
		OSRelease:       osRelease,
		Architecture:    runtime.GOARCH,
	}
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

	log.Printf("[%s] Successfully sent metrics (CPU: %.1f%%, Memory: %.1f%%, Disk: %.1f%%, Processes: %d, Docker: %d)", 
		a.agentID, metrics.CPUUsage, metrics.Memory.UsedPercent, metrics.Disk.UsedPercent, 
		len(metrics.Processes), len(metrics.DockerStats))
	return nil
}
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"errors"
	"net/http"


	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Stats represents Docker container statistics
type Stats struct {
	CPUStats    CPUStats    `json:"cpu_stats"`
	PreCPUStats CPUStats    `json:"precpu_stats"`
	MemoryStats MemoryStats `json:"memory_stats"`
	Networks    map[string]NetworkStats `json:"networks"`
	BlkioStats  BlkioStats  `json:"blkio_stats"`
	PidsStats   PidsStats   `json:"pids_stats"`
}

type CPUStats struct {
	CPUUsage       CPUUsage `json:"cpu_usage"`
	SystemUsage    uint64   `json:"system_cpu_usage"`
	OnlineCPUs     uint32   `json:"online_cpus"`
}

type CPUUsage struct {
	TotalUsage  uint64   `json:"total_usage"`
	PercpuUsage []uint64 `json:"percpu_usage"`
}

type MemoryStats struct {
	Usage   uint64 `json:"usage"`
	Limit   uint64 `json:"limit"`
}

type NetworkStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type BlkioStats struct {
	IoServiceBytesRecursive []BlkioEntry `json:"io_service_bytes_recursive"`
}

type BlkioEntry struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

type PidsStats struct {
	Current uint64 `json:"current"`
}

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
	ContainerID   string `json:"container_id"`
	Name          string `json:"name"`
	CPUPercent    string `json:"cpu_percent"`
	MemoryUsage   string `json:"memory_usage"`
	MemoryLimit   string `json:"memory_limit"`
	MemoryPercent string `json:"memory_percent"`
	NetworkIO     string `json:"network_io"`
	BlockIO       string `json:"block_io"`
	PIDs          string `json:"pids"`
}

// SystemInfo holds additional system information
type SystemInfo struct {
	TotalProcesses  int    `json:"total_processes"`
	DockerAvailable bool   `json:"docker_available"`
	KernelVersion   string `json:"kernel_version"`
	OSRelease       string `json:"os_release"`
	Architecture    string `json:"architecture"`
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
	AgentVersion          = "1.1.0"
	DefaultMetricsInterval = 10 * time.Second
	DefaultRetryInterval   = 5 * time.Second
	MaxRetryAttempts      = 5
	HTTPTimeout           = 30 * time.Second
	MaxProcesses          = 50
)

type Agent struct {
	agentID       string
	homeServerURL string
	client        *http.Client
	hostname      string
	retryCount    int
	startTime     time.Time
	hostProc      string
	hostSys       string
	hostRoot      string
	dockerClient  *client.Client
	logger        *slog.Logger
	mu            sync.Mutex
}

func NewAgent(homeServerURL string) (*Agent, error) {
	if homeServerURL == "" {
		return nil, fmt.Errorf("HOME_SERVER_URL environment variable is required")
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = hostname
	}
	if agentName := os.Getenv("AGENT_NAME"); agentName != "" {
		agentID = agentName
	}

	hostProc := os.Getenv("HOST_PROC")
	if hostProc == "" {
		hostProc = "/proc"
	}

	hostSys := os.Getenv("HOST_SYS")
	if hostSys == "" {
		hostSys = "/sys"
	}

	hostRoot := os.Getenv("HOST_ROOT")
	if hostRoot == "" {
		hostRoot = "/"
	}

	var dockerClient *client.Client
	if dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
		dockerClient = dc
	} else {
		slog.Warn("Failed to create Docker client", "error", err)
	}

	return &Agent{
		agentID:       agentID,
		homeServerURL: strings.TrimSuffix(homeServerURL, "/"),
		client: &http.Client{
			Timeout: HTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		hostname:     hostname,
		startTime:    time.Now(),
		hostProc:     hostProc,
		hostSys:      hostSys,
		hostRoot:     hostRoot,
		dockerClient: dockerClient,
		logger:       slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}, nil
}

func main() {
	homeServerURL := os.Getenv("HOME_SERVER_URL")
	agent, err := NewAgent(homeServerURL)
	if err != nil {
		slog.Error("Failed to create agent", "error", err)
		os.Exit(1)
	}

	agent.logger.Info("Starting VPS Monitor Agent",
		"version", AgentVersion,
		"agent_id", agent.agentID,
		"hostname", agent.hostname,
		"host_proc", agent.hostProc,
		"home_server", homeServerURL)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go agent.collectAndSendMetrics(ctx)

	<-ctx.Done()
	agent.logger.Info("Shutting down agent")
	if agent.dockerClient != nil {
		agent.dockerClient.Close()
	}
	agent.client.CloseIdleConnections()
	time.Sleep(2 * time.Second)
	agent.logger.Info("Agent stopped", "agent_id", agent.agentID)
}

func (a *Agent) collectAndSendMetrics(ctx context.Context) {
	ticker := time.NewTicker(DefaultMetricsInterval)
	defer ticker.Stop()

	if err := a.sendMetricsWithRetry(ctx); err != nil {
		a.logger.Error("Initial metrics send failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.sendMetricsWithRetry(ctx); err != nil {
				a.logger.Error("Metrics send failed", "error", err)
			}
		}
	}
}

func (a *Agent) sendMetricsWithRetry(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for attempt := 1; attempt <= MaxRetryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		metrics, err := a.collectMetrics()
		if err != nil {
			a.logger.Error("Failed to collect metrics", "attempt", attempt, "error", err)
			if attempt < MaxRetryAttempts {
				time.Sleep(DefaultRetryInterval)
				continue
			}
			return err
		}

		if err := a.sendMetrics(metrics); err != nil {
			a.logger.Error("Failed to send metrics", "attempt", attempt, "error", err)
			if attempt < MaxRetryAttempts {
				time.Sleep(DefaultRetryInterval)
				continue
			}
			a.retryCount++
			return err
		}

		if a.retryCount > 0 {
			a.logger.Info("Reconnected successfully", "failed_attempts", a.retryCount)
			a.retryCount = 0
		}
		return nil
	}
	return fmt.Errorf("max retry attempts reached")
}

func (a *Agent) collectMetrics() (*Metrics, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var metrics Metrics
	var collectErr error

	collect := func(fn func() error) {
		defer wg.Add(-1)
		if err := fn(); err != nil {
			mu.Lock()
			collectErr = errors.Join(collectErr, err)
			mu.Unlock()
		}
	}

	wg.Add(5)
	go collect(func() error {
		hostInfo, err := host.Info()
		if err != nil {
			return fmt.Errorf("failed to get host info: %w", err)
		}
		metrics.Hostname = a.hostname
		metrics.Uptime = hostInfo.Uptime
		return nil
	})

	go collect(func() error {
		cpuUsage, err := cpu.Percent(time.Second, false)
		if err != nil {
			return fmt.Errorf("failed to get CPU usage: %w", err)
		}
		metrics.CPUUsage = cpuUsage[0]
		return nil
	})

	go collect(func() error {
		memory, err := mem.VirtualMemory()
		if err != nil {
			return fmt.Errorf("failed to get memory info: %w", err)
		}
		metrics.Memory = memory
		return nil
	})

	go collect(func() error {
		diskUsage, err := disk.Usage(a.hostRoot)
		if err != nil {
			return fmt.Errorf("failed to get disk usage: %w", err)
		}
		metrics.Disk = diskUsage
		return nil
	})

	go collect(func() error {
		network, err := net.IOCounters(true)
		if err != nil {
			a.logger.Warn("Failed to get network info", "error", err)
			metrics.Network = []net.IOCountersStat{}
			return nil
		}
		metrics.Network = network
		return nil
	})

	wg.Wait()
	if collectErr != nil {
		return nil, collectErr
	}

	metrics.AgentID = a.agentID
	metrics.Processes, _ = a.getProcessList()
	metrics.DockerStats = a.getDockerStats()
	metrics.SystemInfo = a.getSystemInfo()
	metrics.AgentInfo = a.getAgentInfo()

	return &metrics, nil
}

func (a *Agent) getProcessList() ([]*ProcessInfo, error) {
	var processes []*ProcessInfo

	if a.isInContainer() && a.hostProc != "/proc" {
		if procs := a.getProcessesFromHostProc(); len(procs) > 0 {
			processes = procs
		}
	} else if procs := a.getProcessesFromPS(); len(procs) > 0 {
		processes = procs
	} else {
		var err error
		if processes, err = a.getProcessesFromPsutil(); err != nil {
			return nil, err
		}
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})

	if len(processes) > 30 {
		processes = processes[:30]
	}

	return processes, nil
}

func (a *Agent) getProcessesFromHostProc() []*ProcessInfo {
	procDirs, err := os.ReadDir(a.hostProc)
	if err != nil {
		a.logger.Warn("Failed to read host proc directory", "error", err)
		return nil
	}

	var processes []*ProcessInfo
	var wg sync.WaitGroup
	procChan := make(chan *ProcessInfo, MaxProcesses)

	for _, dir := range procDirs {
		if len(processes) >= MaxProcesses {
			break
		}

		if !dir.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(dir.Name())
		if err != nil {
			continue
		}

		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			if proc, err := a.readProcessFromProc(int32(pid)); err == nil {
				procChan <- proc
			}
		}(pid)
	}

	go func() {
		wg.Wait()
		close(procChan)
	}()

	for proc := range procChan {
		processes = append(processes, proc)
	}

	a.logger.Info("Collected processes from host proc", "count", len(processes))
	return processes
}

func (a *Agent) readProcessFromProc(pid int32) (*ProcessInfo, error) {
	procPath := filepath.Join(a.hostProc, strconv.Itoa(int(pid)))

	commData, err := os.ReadFile(filepath.Join(procPath, "comm"))
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(string(commData))

	cmdlineData, err := os.ReadFile(filepath.Join(procPath, "cmdline"))
	command := name
	if err == nil {
		command = strings.TrimSpace(strings.ReplaceAll(string(cmdlineData), "\x00", " "))
	}

	memoryPercent := float32(0)
	memoryMB := float64(0)
	if statusData, err := os.ReadFile(filepath.Join(procPath, "status")); err == nil {
		lines := strings.Split(string(statusData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if rss, err := strconv.ParseFloat(fields[1], 64); err == nil {
						memoryMB = rss / 1024 // Convert KB to MB
						if totalMem, err := mem.VirtualMemory(); err == nil && totalMem.Total > 0 {
							memoryPercent = float32((rss * 1024) / float64(totalMem.Total) * 100)
						}
					}
				}
				break
			}
		}
	}

	return &ProcessInfo{
		PID:           pid,
		Name:          name,
		CPUPercent:    0,
		MemoryPercent: memoryPercent,
		MemoryMB:      memoryMB,
		Command:       command,
	}, nil
}

func (a *Agent) isInContainer() bool {
	if a.hostProc != "/proc" {
		return true
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		cgroupContent := string(data)
		return strings.Contains(cgroupContent, "docker") ||
			strings.Contains(cgroupContent, "containerd") ||
			strings.Contains(cgroupContent, "kubepods")
	}
	return false
}

func (a *Agent) getProcessesFromPS() []*ProcessInfo {
	cmd := exec.Command("ps", "aux", "--no-headers")
	if a.hostRoot != "/" && a.isInContainer() {
		cmd = exec.Command("chroot", a.hostRoot, "ps", "aux", "--no-headers")
	}

	output, err := cmd.Output()
	if err != nil {
		a.logger.Warn("Failed to run ps command", "error", err)
		return nil
	}

	var processes []*ProcessInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			continue
		}

		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memPercent, _ := strconv.ParseFloat(fields[3], 32)
		name := fields[10]
		if len(name) > 50 {
			name = name[:50] + "..."
		}

		processes = append(processes, &ProcessInfo{
			PID:           int32(pid),
			Name:          name,
			CPUPercent:    cpuPercent,
			MemoryPercent: float32(memPercent),
			Command:       strings.Join(fields[10:], " "),
		})
	}

	a.logger.Info("Collected processes from ps", "count", len(processes))
	return processes
}

func (a *Agent) getProcessesFromPsutil() ([]*ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []*ProcessInfo
	var wg sync.WaitGroup
	procChan := make(chan *ProcessInfo, MaxProcesses)

	for _, pid := range pids {
		if len(processes) >= MaxProcesses {
			break
		}

		wg.Add(1)
		go func(pid int32) {
			defer wg.Done()
			if p, err := process.NewProcess(pid); err == nil {
				name, _ := p.Name()
				cpuPercent, _ := p.CPUPercent()
				memPercent, _ := p.MemoryPercent()
				memInfo, _ := p.MemoryInfo()
				memoryMB := float64(0)
				if memInfo != nil {
					memoryMB = float64(memInfo.RSS) / 1024 / 1024
				}
				cmdline, _ := p.Cmdline()

				procChan <- &ProcessInfo{
					PID:           pid,
					Name:          name,
					CPUPercent:    cpuPercent,
					MemoryPercent: memPercent,
					MemoryMB:      memoryMB,
					Command:       cmdline,
				}
			}
		}(pid)
	}

	go func() {
		wg.Wait()
		close(procChan)
	}()

	for proc := range procChan {
		processes = append(processes, proc)
	}

	a.logger.Info("Collected processes from gopsutil", "count", len(processes))
	return processes, nil
}

func (a *Agent) getDockerStats() []*DockerContainerStat {
	if a.dockerClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := a.dockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		a.logger.Warn("Failed to list containers", "error", err)
		return nil
	}

	var dockerStats []*DockerContainerStat
	var wg sync.WaitGroup
	statsChan := make(chan *DockerContainerStat, len(containers))

	for _, containerInfo := range containers {
		wg.Add(1)
		go func(containerInfo types.Container) {
			defer wg.Done()
			stats, err := a.dockerClient.ContainerStats(ctx, containerInfo.ID, false)
			if err != nil {
				a.logger.Warn("Failed to get container stats", "container_id", containerInfo.ID[:12], "error", err)
				return
			}
			defer stats.Body.Close()

			var containerStats Stats
			if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
				a.logger.Warn("Failed to decode container stats", "container_id", containerInfo.ID[:12], "error", err)
				return
			}

			cpuPercent := calculateCPUPercent(&containerStats)
			memUsage := containerStats.MemoryStats.Usage
			memLimit := containerStats.MemoryStats.Limit
			memPercent := float64(0)
			if memLimit > 0 {
				memPercent = (float64(memUsage) / float64(memLimit)) * 100.0
			}

			var rxBytes, txBytes uint64
			for _, network := range containerStats.Networks {
				rxBytes += network.RxBytes
				txBytes += network.TxBytes
			}
			networkIO := fmt.Sprintf("%s / %s", formatBytes(rxBytes), formatBytes(txBytes))

			var readBytes, writeBytes uint64
			for _, blkio := range containerStats.BlkioStats.IoServiceBytesRecursive {
				if blkio.Op == "Read" {
					readBytes += blkio.Value
				} else if blkio.Op == "Write" {
					writeBytes += blkio.Value
				}
			}
			blockIO := fmt.Sprintf("%s / %s", formatBytes(readBytes), formatBytes(writeBytes))

			containerName := containerInfo.Names[0]
			if strings.HasPrefix(containerName, "/") {
				containerName = containerName[1:]
			}

			dockerStat := &DockerContainerStat{
				ContainerID:   containerInfo.ID[:12],
				Name:          containerName,
				CPUPercent:    fmt.Sprintf("%.2f%%", cpuPercent),
				MemoryUsage:   formatBytes(memUsage),
				MemoryLimit:   formatBytes(memLimit),
				MemoryPercent: fmt.Sprintf("%.2f%%", memPercent),
				NetworkIO:     networkIO,
				BlockIO:       blockIO,
				PIDs:          fmt.Sprintf("%d", containerStats.PidsStats.Current),
			}

			statsChan <- dockerStat
			a.logger.Info("Collected container stats",
				"container_name", containerName,
				"cpu_percent", dockerStat.CPUPercent,
				"memory_percent", dockerStat.MemoryPercent)
		}(containerInfo)
	}

	go func() {
		wg.Wait()
		close(statsChan)
	}()

	for stat := range statsChan {
		dockerStats = append(dockerStats, stat)
	}

	a.logger.Info("Collected container stats", "count", len(dockerStats))
	return dockerStats
}

func calculateCPUPercent(stats *Stats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return 0
}

func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return "0B"
	}

	const unit = 1024
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}

func (a *Agent) getSystemInfo() *SystemInfo {
	totalProcesses := 0
	if procDirs, err := os.ReadDir(a.hostProc); err == nil {
		for _, dir := range procDirs {
			if dir.IsDir() {
				if _, err := strconv.Atoi(dir.Name()); err == nil {
					totalProcesses++
				}
			}
		}
	}

	dockerAvailable := a.dockerClient != nil
	if dockerAvailable {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := a.dockerClient.Ping(ctx); err != nil {
			dockerAvailable = false
			a.logger.Warn("Docker API ping failed", "error", err)
		}
	}

	kernelVersion := ""
	if output, err := exec.Command("uname", "-r").Output(); err == nil {
		kernelVersion = strings.TrimSpace(string(output))
	}

	osRelease := ""
	osReleasePath := filepath.Join(a.hostRoot, "/etc/os-release")
	if data, err := os.ReadFile(osReleasePath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
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

func (a *Agent) getAgentInfo() *AgentInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &AgentInfo{
		Version:       AgentVersion,
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemStats: map[string]uint64{
			"alloc":       memStats.Alloc,
			"total_alloc": memStats.TotalAlloc,
			"sys":         memStats.Sys,
			"num_gc":      uint64(memStats.NumGC),
		},
		StartTime: a.startTime,
	}
}

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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	a.logger.Info("Successfully sent metrics",
		"agent_id", a.agentID,
		"cpu_percent", fmt.Sprintf("%.1f", metrics.CPUUsage),
		"memory_percent", fmt.Sprintf("%.1f", metrics.Memory.UsedPercent),
		"disk_percent", fmt.Sprintf("%.1f", metrics.Disk.UsedPercent),
		"process_count", len(metrics.Processes),
		"container_count", len(metrics.DockerStats))
	return nil
}
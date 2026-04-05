package system

import (
	"context"
	"os"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemStats struct {
	HostInfo HostInfo `json:"hostInfo"`
	Usage    Usage    `json:"usage"`
}

type HostInfo struct {
	Hostname        string `json:"hostname"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	KernelVersion   string `json:"kernelVersion"`
	Arch            string `json:"arch"`
	Uptime          uint64 `json:"uptime"`
	CPULogical      int    `json:"cpuLogical"`
	CPUPhysical     int    `json:"cpuPhysical,omitempty"`
}

type Usage struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	MemoryTotal   uint64  `json:"memoryTotal"`
	MemoryUsed    uint64  `json:"memoryUsed"`
	DiskPercent   float64 `json:"diskPercent"`
	DiskTotal     uint64  `json:"diskTotal"`
	DiskUsed      uint64  `json:"diskUsed"`
}

var (
	cachedCPUMutex    sync.Mutex
	cachedCPULogical  int
	cachedCPUPhysical int
)

// Init configures gopsutil to use the host's /proc directory if mounted
func Init() {
	// If we are running in a container and have mounted /proc to /host/proc,
	// we need to tell gopsutil to look there.
	if _, err := os.Stat("/host/proc"); err == nil {
		os.Setenv("HOST_PROC", "/host/proc")
	}
}


func GetStats(ctx context.Context) (*SystemStats, error) {
	hInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get Memory Info
	vMem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get CPU Percent (over a small interval)
	// Note: 0 interval returns immediate value since last call, which might be 0 on first call
	// For a dashboard, we usually want the average over the last second, but that blocks.
	// A better approach for an API is to return the value since last call or just immediate.
	cpuPercents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return nil, err
	}

	var cpuPercent float64
	if len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	cachedCPUMutex.Lock()
	needsFetch := cachedCPULogical == 0
	cachedCPUMutex.Unlock()

	if needsFetch {
		logical, err := cpu.CountsWithContext(ctx, true)
		physical, errPhys := cpu.CountsWithContext(ctx, false)
		
		cachedCPUMutex.Lock()
		if cachedCPULogical == 0 && err == nil && errPhys == nil && logical > 0 {
			cachedCPULogical = logical
			cachedCPUPhysical = physical
		}
		cachedCPUMutex.Unlock()
	}

	cachedCPUMutex.Lock()
	cpuLog := cachedCPULogical
	cpuPhys := cachedCPUPhysical
	cachedCPUMutex.Unlock()

	// Get Disk Usage for root partition
	// If running in container with /host mounted, use /host, otherwise use /
	diskPath := "/"
	if _, err := os.Stat("/host"); err == nil {
		diskPath = "/host"
	}

	var diskPercent float64
	var diskTotal, diskUsed uint64
	diskUsage, err := disk.UsageWithContext(ctx, diskPath)
	if err == nil {
		diskPercent = diskUsage.UsedPercent
		diskTotal = diskUsage.Total
		diskUsed = diskUsage.Used
	}

	return &SystemStats{
		HostInfo: HostInfo{
			Hostname:        hInfo.Hostname,
			Platform:        hInfo.Platform,
			PlatformVersion: hInfo.PlatformVersion,
			KernelVersion:   hInfo.KernelVersion,
			Arch:            runtime.GOARCH,
			Uptime:          hInfo.Uptime,
			CPULogical:      cpuLog,
			CPUPhysical:     cpuPhys,
		},
		Usage: Usage{
			CPUPercent:    cpuPercent,
			MemoryPercent: vMem.UsedPercent,
			MemoryTotal:   vMem.Total,
			MemoryUsed:    vMem.Used,
			DiskPercent:   diskPercent,
			DiskTotal:     diskTotal,
			DiskUsed:      diskUsed,
		},
	}, nil
}

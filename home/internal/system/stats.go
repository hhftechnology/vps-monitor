package system

import (
	"context"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
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
}

type Usage struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	MemoryTotal   uint64  `json:"memoryTotal"`
	MemoryUsed    uint64  `json:"memoryUsed"`
}

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
	// Let's use a very short interval for responsiveness, or 0.
	cpuPercents, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return nil, err
	}

	var cpuPercent float64
	if len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	return &SystemStats{
		HostInfo: HostInfo{
			Hostname:        hInfo.Hostname,
			Platform:        hInfo.Platform,
			PlatformVersion: hInfo.PlatformVersion,
			KernelVersion:   hInfo.KernelVersion,
			Arch:            runtime.GOARCH,
			Uptime:          hInfo.Uptime,
		},
		Usage: Usage{
			CPUPercent:    cpuPercent,
			MemoryPercent: vMem.UsedPercent,
			MemoryTotal:   vMem.Total,
			MemoryUsed:    vMem.Used,
		},
	}, nil
}

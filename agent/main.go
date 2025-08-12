// agent/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	Hostname      string        `json:"hostname"`
	Uptime        uint64        `json:"uptime"`
	CPUUsage      float64       `json:"cpu_usage"`
	Memory        *mem.VirtualMemoryStat `json:"memory"`
	Disk          *disk.UsageStat        `json:"disk"`
	Network       []net.IOCountersStat   `json:"network"`
	Processes     []*ProcessInfo `json:"processes"`
}

// ProcessInfo holds simplified information about a running process.
type ProcessInfo struct {
	PID         int32   `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
}

func main() {
	homeServerURL := os.Getenv("HOME_SERVER_URL")
	if homeServerURL == "" {
		log.Fatal("HOME_SERVER_URL environment variable is not set.")
	}
	log.Printf("Starting agent. Will report to: %s", homeServerURL)

	// Send metrics every 10 seconds.
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics, err := collectMetrics()
		if err != nil {
			log.Printf("Error collecting metrics: %v", err)
			continue
		}

		if err := sendMetrics(homeServerURL, metrics); err != nil {
			log.Printf("Error sending metrics: %v", err)
		}
	}
}

// collectMetrics gathers all the system data.
func collectMetrics() (*Metrics, error) {
	hostname, _ := os.Hostname()
	hostInfo, _ := host.Info()
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	network, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}
    
    processes, err := getProcessList()
    if err != nil {
        return nil, err
    }

	return &Metrics{
		Hostname:      hostname,
		Uptime:        hostInfo.Uptime,
		CPUUsage:      cpuUsage[0],
		Memory:        memory,
		Disk:          diskUsage,
		Network:       network,
        Processes:     processes,
	}, nil
}

// getProcessList gets a list of running processes.
func getProcessList() ([]*ProcessInfo, error) {
    pids, err := process.Pids()
    if err != nil {
        return nil, err
    }

    var processes []*ProcessInfo
    for _, pid := range pids[:30] { // Limit to top 30 processes for brevity
        p, err := process.NewProcess(pid)
        if err != nil {
            continue // Process might have terminated
        }
        name, _ := p.Name()
        cpuPercent, _ := p.CPUPercent()
        memPercent, _ := p.MemoryPercent()

        processes = append(processes, &ProcessInfo{
            PID:         pid,
            Name:        name,
            CPUPercent:  cpuPercent,
            MemoryPercent: memPercent,
        })
    }
    return processes, nil
}


// sendMetrics sends the collected data to the home server.
func sendMetrics(url string, metrics *Metrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url+"/api/metrics", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %d", resp.StatusCode)
	}

	log.Println("Successfully sent metrics to home server.")
	return nil
}

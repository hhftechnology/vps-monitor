package stats

import (
	"sync"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

const defaultMaxDataPoints = 1440

type DataPoint struct {
	CPUPercent    float64
	MemoryPercent float64
	Timestamp     time.Time
}

type ContainerHistory struct {
	dataPoints []DataPoint
}

type HistoryManager struct {
	mu         sync.RWMutex
	containers map[string]*ContainerHistory
	maxSize    int
}

func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		containers: make(map[string]*ContainerHistory),
		maxSize:    defaultMaxDataPoints,
	}
}

func ContainerKey(host, containerID string) string {
	return host + ":" + containerID
}

func (hm *HistoryManager) RecordStats(host, containerID string, stats models.ContainerStats) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	key := ContainerKey(host, containerID)
	history, exists := hm.containers[key]
	if !exists {
		history = &ContainerHistory{
			dataPoints: make([]DataPoint, 0, hm.maxSize),
		}
		hm.containers[key] = history
	}

	history.dataPoints = append(history.dataPoints, DataPoint{
		CPUPercent:    stats.CPUPercent,
		MemoryPercent: stats.MemoryPercent,
		Timestamp:     time.Unix(stats.Timestamp, 0),
	})

	if len(history.dataPoints) > hm.maxSize {
		history.dataPoints = history.dataPoints[len(history.dataPoints)-hm.maxSize:]
	}
}

func (hm *HistoryManager) GetAverages(host, containerID string, duration time.Duration) (cpuAvg, memAvg float64, hasData bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	history, exists := hm.containers[ContainerKey(host, containerID)]
	if !exists || len(history.dataPoints) == 0 {
		return 0, 0, false
	}

	cutoff := time.Now().Add(-duration)
	var cpuSum, memSum float64
	var count int

	for i := len(history.dataPoints) - 1; i >= 0; i-- {
		point := history.dataPoints[i]
		if point.Timestamp.Before(cutoff) {
			break
		}
		cpuSum += point.CPUPercent
		memSum += point.MemoryPercent
		count++
	}

	if count == 0 {
		return 0, 0, false
	}

	return cpuSum / float64(count), memSum / float64(count), true
}

func (hm *HistoryManager) Get1hAverages(host, containerID string) (cpuAvg, memAvg float64, hasData bool) {
	return hm.GetAverages(host, containerID, time.Hour)
}

func (hm *HistoryManager) Get12hAverages(host, containerID string) (cpuAvg, memAvg float64, hasData bool) {
	return hm.GetAverages(host, containerID, 12*time.Hour)
}

func (hm *HistoryManager) CleanupContainer(host, containerID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.containers, ContainerKey(host, containerID))
}

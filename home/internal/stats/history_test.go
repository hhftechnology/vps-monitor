package stats

import (
	"testing"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func TestHistoryManagerUsesHostScopedKeys(t *testing.T) {
	manager := NewHistoryManager()
	now := time.Now().Unix()

	manager.RecordStats("host-a", "container-1", models.ContainerStats{
		CPUPercent:    10,
		MemoryPercent: 20,
		Timestamp:     now,
	})
	manager.RecordStats("host-b", "container-1", models.ContainerStats{
		CPUPercent:    50,
		MemoryPercent: 60,
		Timestamp:     now,
	})

	cpuA, memA, okA := manager.Get1hAverages("host-a", "container-1")
	cpuB, memB, okB := manager.Get1hAverages("host-b", "container-1")

	if !okA || !okB {
		t.Fatalf("expected both host/container pairs to have data")
	}
	if cpuA != 10 || memA != 20 {
		t.Fatalf("unexpected host-a averages: cpu=%v mem=%v", cpuA, memA)
	}
	if cpuB != 50 || memB != 60 {
		t.Fatalf("unexpected host-b averages: cpu=%v mem=%v", cpuB, memB)
	}
}

func TestHistoryManagerExcludesExpiredPoints(t *testing.T) {
	manager := NewHistoryManager()

	manager.RecordStats("host-a", "container-1", models.ContainerStats{
		CPUPercent:    90,
		MemoryPercent: 90,
		Timestamp:     time.Now().Add(-13 * time.Hour).Unix(),
	})
	manager.RecordStats("host-a", "container-1", models.ContainerStats{
		CPUPercent:    30,
		MemoryPercent: 40,
		Timestamp:     time.Now().Unix(),
	})

	cpu, mem, ok := manager.Get12hAverages("host-a", "container-1")
	if !ok {
		t.Fatal("expected in-window data")
	}
	if cpu != 30 || mem != 40 {
		t.Fatalf("expected only recent sample, got cpu=%v mem=%v", cpu, mem)
	}
}

package scanner

import (
	"testing"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func TestNewScanDBCreatesContainerStatsTable(t *testing.T) {
	db := newTestScanDB(t)

	var tableName string
	if err := db.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'container_stats'`,
	).Scan(&tableName); err != nil {
		t.Fatalf("expected container_stats table to exist: %v", err)
	}

	if tableName != "container_stats" {
		t.Fatalf("unexpected table name: %q", tableName)
	}
}

func TestContainerStatsAveragesUseConfiguredWindows(t *testing.T) {
	db := newTestScanDB(t)
	now := time.Unix(1_700_000_000, 0).UTC()

	samples := []models.ContainerStats{
		{
			ContainerID:   "container-1",
			Host:          "host-a",
			CPUPercent:    90,
			MemoryPercent: 95,
			Timestamp:     now.Add(-13 * time.Hour).Unix(),
		},
		{
			ContainerID:   "container-1",
			Host:          "host-a",
			CPUPercent:    40,
			MemoryPercent: 60,
			Timestamp:     now.Add(-2 * time.Hour).Unix(),
		},
		{
			ContainerID:   "container-1",
			Host:          "host-a",
			CPUPercent:    10,
			MemoryPercent: 20,
			Timestamp:     now.Add(-30 * time.Minute).Unix(),
		},
	}

	for _, sample := range samples {
		if err := db.InsertContainerStat(sample); err != nil {
			t.Fatalf("InsertContainerStat() error = %v", err)
		}
	}

	history, err := db.GetContainerHistoricalAverages("host-a", "container-1", now)
	if err != nil {
		t.Fatalf("GetContainerHistoricalAverages() error = %v", err)
	}

	if !history.HasData {
		t.Fatal("expected historical data")
	}
	if history.CPU1h != 10 || history.Memory1h != 20 {
		t.Fatalf("unexpected 1h averages: cpu=%v memory=%v", history.CPU1h, history.Memory1h)
	}
	if history.CPU12h != 25 || history.Memory12h != 40 {
		t.Fatalf("unexpected 12h averages: cpu=%v memory=%v", history.CPU12h, history.Memory12h)
	}
}

func TestGetRecentContainerStatsReturnsAscendingSeries(t *testing.T) {
	db := newTestScanDB(t)
	now := time.Unix(1_700_000_000, 0).UTC()

	for _, sample := range []models.ContainerStats{
		{ContainerID: "container-1", Host: "host-a", CPUPercent: 10, Timestamp: now.Add(-3 * time.Minute).Unix()},
		{ContainerID: "container-1", Host: "host-a", CPUPercent: 20, Timestamp: now.Add(-2 * time.Minute).Unix()},
		{ContainerID: "container-1", Host: "host-a", CPUPercent: 30, Timestamp: now.Add(-1 * time.Minute).Unix()},
	} {
		if err := db.InsertContainerStat(sample); err != nil {
			t.Fatalf("InsertContainerStat() error = %v", err)
		}
	}

	series, err := db.GetRecentContainerStats("host-a", "container-1", now.Add(-12*time.Hour), 2)
	if err != nil {
		t.Fatalf("GetRecentContainerStats() error = %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(series))
	}
	if series[0].CPUPercent != 20 || series[1].CPUPercent != 30 {
		t.Fatalf("expected ascending latest samples, got %+v", series)
	}
	if series[0].Timestamp >= series[1].Timestamp {
		t.Fatalf("expected ascending timestamps, got %d then %d", series[0].Timestamp, series[1].Timestamp)
	}
}

func TestPruneContainerStatsOlderThanRemovesExpiredRows(t *testing.T) {
	db := newTestScanDB(t)
	now := time.Unix(1_700_000_000, 0).UTC()

	oldSample := models.ContainerStats{
		ContainerID:   "container-1",
		Host:          "host-a",
		CPUPercent:    10,
		MemoryPercent: 10,
		Timestamp:     now.Add(-31 * 24 * time.Hour).Unix(),
	}
	newSample := models.ContainerStats{
		ContainerID:   "container-1",
		Host:          "host-a",
		CPUPercent:    20,
		MemoryPercent: 20,
		Timestamp:     now.Add(-1 * time.Hour).Unix(),
	}

	if err := db.InsertContainerStat(oldSample); err != nil {
		t.Fatalf("InsertContainerStat(oldSample) error = %v", err)
	}
	if err := db.InsertContainerStat(newSample); err != nil {
		t.Fatalf("InsertContainerStat(newSample) error = %v", err)
	}

	if err := db.PruneContainerStatsOlderThan(now.Add(-30 * 24 * time.Hour)); err != nil {
		t.Fatalf("PruneContainerStatsOlderThan() error = %v", err)
	}

	series, err := db.GetRecentContainerStats("host-a", "container-1", now.Add(-90*24*time.Hour), 10)
	if err != nil {
		t.Fatalf("GetRecentContainerStats() error = %v", err)
	}

	if len(series) != 1 {
		t.Fatalf("expected 1 sample after pruning, got %d", len(series))
	}
	if series[0].Timestamp != newSample.Timestamp {
		t.Fatalf("expected recent sample to remain, got %+v", series[0])
	}
}

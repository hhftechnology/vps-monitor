package scanner

import (
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func newTestScanResultStore(t *testing.T) *ScanResultStore {
	t.Helper()
	return NewScanResultStore(newTestScanDB(t))
}

func TestScanResultStoreAddAndGetResults(t *testing.T) {
	store := newTestScanResultStore(t)

	result := models.ScanResult{
		ID:       "result-1",
		ImageRef: "nginx:latest",
		Host:     "local",
		Scanner:  models.ScannerGrype,
		Summary:  models.SeveritySummary{Total: 2, High: 2},
	}

	if err := store.Add(result); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	results := store.GetResults("local", "nginx:latest")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "result-1" {
		t.Fatalf("expected result ID 'result-1', got %q", results[0].ID)
	}
}

func TestScanResultStoreGetResultsEmptyForUnknownImage(t *testing.T) {
	store := newTestScanResultStore(t)

	results := store.GetResults("local", "nonexistent:latest")
	if results == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestScanResultStoreGetLatestReturnsNewest(t *testing.T) {
	store := newTestScanResultStore(t)

	older := models.ScanResult{
		ID:          "old",
		ImageRef:    "alpine:3.18",
		Host:        "local",
		Scanner:     models.ScannerGrype,
		CompletedAt: 100,
	}
	newer := models.ScanResult{
		ID:          "new",
		ImageRef:    "alpine:3.18",
		Host:        "local",
		Scanner:     models.ScannerGrype,
		CompletedAt: 200,
	}

	if err := store.Add(older); err != nil {
		t.Fatalf("Add(older) error = %v", err)
	}
	if err := store.Add(newer); err != nil {
		t.Fatalf("Add(newer) error = %v", err)
	}

	latest := store.GetLatest("local", "alpine:3.18")
	if latest == nil {
		t.Fatal("expected a result, got nil")
	}
	if latest.ID != "new" {
		t.Fatalf("expected newest result ID 'new', got %q", latest.ID)
	}
}

func TestScanResultStoreGetLatestReturnsNilWhenEmpty(t *testing.T) {
	store := newTestScanResultStore(t)

	result := store.GetLatest("local", "missing:image")
	if result != nil {
		t.Fatalf("expected nil for unknown image, got %+v", result)
	}
}

func TestScanResultStoreIsolatesByHost(t *testing.T) {
	store := newTestScanResultStore(t)

	if err := store.Add(models.ScanResult{ID: "host-a", ImageRef: "redis:7", Host: "host-a", Scanner: models.ScannerGrype}); err != nil {
		t.Fatalf("Add(host-a) error = %v", err)
	}
	if err := store.Add(models.ScanResult{ID: "host-b", ImageRef: "redis:7", Host: "host-b", Scanner: models.ScannerGrype}); err != nil {
		t.Fatalf("Add(host-b) error = %v", err)
	}

	resultsA := store.GetResults("host-a", "redis:7")
	resultsB := store.GetResults("host-b", "redis:7")

	if len(resultsA) != 1 || resultsA[0].ID != "host-a" {
		t.Fatalf("expected 1 result for host-a, got %+v", resultsA)
	}
	if len(resultsB) != 1 || resultsB[0].ID != "host-b" {
		t.Fatalf("expected 1 result for host-b, got %+v", resultsB)
	}
}

func TestScanResultStoreDBAccessor(t *testing.T) {
	store := newTestScanResultStore(t)

	if store.DB() == nil {
		t.Fatal("expected non-nil DB accessor")
	}
}

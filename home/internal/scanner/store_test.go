package scanner

import (
	"fmt"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func TestScanResultStoreAddAndGetResults(t *testing.T) {
	store := NewScanResultStore()

	result := models.ScanResult{
		ID:       "result-1",
		ImageRef: "nginx:latest",
		Host:     "local",
		Scanner:  models.ScannerGrype,
		Summary:  models.SeveritySummary{Total: 2, High: 2},
	}

	store.Add(result)

	results := store.GetResults("local", "nginx:latest")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "result-1" {
		t.Fatalf("expected result ID 'result-1', got %q", results[0].ID)
	}
}

func TestScanResultStoreGetResultsEmptyForUnknownImage(t *testing.T) {
	store := NewScanResultStore()

	results := store.GetResults("local", "nonexistent:latest")
	if results == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestScanResultStoreGetLatestReturnsNewest(t *testing.T) {
	store := NewScanResultStore()

	older := models.ScanResult{ID: "old", ImageRef: "alpine:3.18", Host: "local", CompletedAt: 100}
	newer := models.ScanResult{ID: "new", ImageRef: "alpine:3.18", Host: "local", CompletedAt: 200}

	store.Add(older)
	store.Add(newer)

	latest := store.GetLatest("local", "alpine:3.18")
	if latest == nil {
		t.Fatal("expected a result, got nil")
	}
	// Add prepends, so the last-added is first
	if latest.ID != "new" {
		t.Fatalf("expected newest result ID 'new', got %q", latest.ID)
	}
}

func TestScanResultStoreGetLatestReturnsNilWhenEmpty(t *testing.T) {
	store := NewScanResultStore()

	result := store.GetLatest("local", "missing:image")
	if result != nil {
		t.Fatalf("expected nil for unknown image, got %+v", result)
	}
}

func TestScanResultStoreIsolatesByHost(t *testing.T) {
	store := NewScanResultStore()

	store.Add(models.ScanResult{ID: "host-a", ImageRef: "redis:7", Host: "host-a"})
	store.Add(models.ScanResult{ID: "host-b", ImageRef: "redis:7", Host: "host-b"})

	resultsA := store.GetResults("host-a", "redis:7")
	resultsB := store.GetResults("host-b", "redis:7")

	if len(resultsA) != 1 || resultsA[0].ID != "host-a" {
		t.Fatalf("expected 1 result for host-a, got %+v", resultsA)
	}
	if len(resultsB) != 1 || resultsB[0].ID != "host-b" {
		t.Fatalf("expected 1 result for host-b, got %+v", resultsB)
	}
}

func TestScanResultStoreCapsAtMaxResults(t *testing.T) {
	store := NewScanResultStore()

	for i := range maxResultsPerImage + 5 {
		store.Add(models.ScanResult{
			ID:       fmt.Sprintf("result-%d", i),
			ImageRef: "ubuntu:22.04",
			Host:     "local",
		})
	}

	results := store.GetResults("local", "ubuntu:22.04")
	if len(results) != maxResultsPerImage {
		t.Fatalf("expected results capped at %d, got %d", maxResultsPerImage, len(results))
	}
}

func TestScanResultStoreGetResultsReturnsCopy(t *testing.T) {
	store := NewScanResultStore()
	store.Add(models.ScanResult{ID: "r1", ImageRef: "img:tag", Host: "local"})

	results := store.GetResults("local", "img:tag")
	results[0].ID = "mutated"

	// Original in store should be unchanged
	originals := store.GetResults("local", "img:tag")
	if originals[0].ID == "mutated" {
		t.Fatal("GetResults should return a copy, not a reference to internal data")
	}
}
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

func TestGetContainerHistoricalStatsReturnsPersistedDataWithoutAlerts(t *testing.T) {
	db := newTestAPIScanDB(t)
	now := time.Now().UTC()

	for _, sample := range []models.ContainerStats{
		{
			ContainerID:   "container-1",
			Host:          "host-a",
			CPUPercent:    12,
			MemoryPercent: 34,
			MemoryUsage:   512,
			MemoryLimit:   1024,
			Timestamp:     now.Add(-30 * time.Minute).Unix(),
		},
		{
			ContainerID:   "container-1",
			Host:          "host-a",
			CPUPercent:    20,
			MemoryPercent: 40,
			MemoryUsage:   768,
			MemoryLimit:   1024,
			Timestamp:     now.Add(-15 * time.Minute).Unix(),
		},
	} {
		if err := db.InsertContainerStat(sample); err != nil {
			t.Fatalf("InsertContainerStat() error = %v", err)
		}
	}

	router := &APIRouter{
		registry: services.NewRegistry(nil, nil, nil, &config.Config{}, nil),
		statsDB:  db,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers/container-1/stats/history?host=host-a", nil)
	req = withURLParam(req, "id", "container-1")
	rec := httptest.NewRecorder()

	router.GetContainerHistoricalStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var history models.HistoricalAverages
	if err := json.Unmarshal(rec.Body.Bytes(), &history); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !history.HasData {
		t.Fatal("expected historical data")
	}
	if len(history.Samples) != 2 {
		t.Fatalf("expected 2 bootstrap samples, got %d", len(history.Samples))
	}
}

func TestEnrichContainersWithHistoricalStatsUsesDatabase(t *testing.T) {
	db := newTestAPIScanDB(t)
	now := time.Now().UTC()

	if err := db.InsertContainerStat(models.ContainerStats{
		ContainerID:   "container-1",
		Host:          "host-a",
		CPUPercent:    18,
		MemoryPercent: 42,
		Timestamp:     now.Add(-20 * time.Minute).Unix(),
	}); err != nil {
		t.Fatalf("InsertContainerStat() error = %v", err)
	}

	router := &APIRouter{statsDB: db}
	containers := []models.ContainerInfo{
		{
			ID:    "container-1",
			Host:  "host-a",
			Names: []string{"/api"},
		},
	}

	router.enrichContainersWithHistoricalStats(containers)

	if containers[0].HistoricalStats == nil {
		t.Fatal("expected historical stats to be attached")
	}
	if containers[0].HistoricalStats.CPU1h != 18 || containers[0].HistoricalStats.Memory1h != 42 {
		t.Fatalf("unexpected historical stats: %+v", containers[0].HistoricalStats)
	}
}

func newTestAPIScanDB(t *testing.T) *scanner.ScanDB {
	t.Helper()

	db, err := scanner.NewScanDB(t.TempDir() + "/scan.db")
	if err != nil {
		t.Fatalf("NewScanDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
}

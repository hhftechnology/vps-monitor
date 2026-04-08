package scanner

import (
	"path/filepath"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func newTestScanDB(t *testing.T) *ScanDB {
	t.Helper()

	db, err := NewScanDB(filepath.Join(t.TempDir(), "scan.db"))
	if err != nil {
		t.Fatalf("NewScanDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func sampleSBOMResult(id, imageRef, host string, format models.SBOMFormat, completedAt int64) models.SBOMResult {
	components := []models.SBOMComponent{
		{Name: "alpine-baselayout", Version: "3.4.3-r1", Type: "library", PURL: "pkg:apk/alpine/alpine-baselayout@3.4.3-r1"},
		{Name: "busybox", Version: "1.36.1-r7", Type: "library", PURL: "pkg:apk/alpine/busybox@1.36.1-r7"},
	}

	return models.SBOMResult{
		ID:             id,
		ImageRef:       imageRef,
		Host:           host,
		Format:         format,
		ComponentCount: len(components),
		FileSize:       2048,
		FilePath:       "/data/sbom/" + id + ".json",
		StartedAt:      completedAt - 5,
		CompletedAt:    completedAt,
		DurationMs:     5000,
		Components:     components,
	}
}

func TestInsertSBOMResultPersistsRowAndComponents(t *testing.T) {
	db := newTestScanDB(t)
	result := sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100)

	if err := db.InsertSBOMResult(result, "sha256:img1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	stored, err := db.GetSBOMResultByID(result.ID)
	if err != nil {
		t.Fatalf("GetSBOMResultByID() error = %v", err)
	}
	if stored == nil {
		t.Fatal("GetSBOMResultByID() = nil, want result")
	}
	if stored.ImageRef != result.ImageRef || stored.Host != result.Host || stored.Format != result.Format {
		t.Fatalf("stored SBOM mismatch: got %+v want %+v", stored, result)
	}
	if len(stored.Components) != len(result.Components) {
		t.Fatalf("stored components = %d, want %d", len(stored.Components), len(result.Components))
	}
}

func TestInsertSBOMResultUpsertsImageSBOMState(t *testing.T) {
	db := newTestScanDB(t)

	first := sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100)
	second := sampleSBOMResult("sbom-2", "alpine:3.18", "local", models.SBOMFormatCycloneDX, 200)

	if err := db.InsertSBOMResult(first, "sha256:first"); err != nil {
		t.Fatalf("InsertSBOMResult(first) error = %v", err)
	}
	if err := db.InsertSBOMResult(second, "sha256:second"); err != nil {
		t.Fatalf("InsertSBOMResult(second) error = %v", err)
	}

	state, err := db.GetImageSBOMState("local", "alpine:3.18")
	if err != nil {
		t.Fatalf("GetImageSBOMState() error = %v", err)
	}
	if state == nil {
		t.Fatal("GetImageSBOMState() = nil, want state")
	}
	if state.ImageID != "sha256:second" || state.LastSBOMID != second.ID || state.LastSBOMAt != second.CompletedAt {
		t.Fatalf("unexpected SBOM state: %+v", state)
	}
}

func TestGetSBOMResultByIDReturnsComponents(t *testing.T) {
	db := newTestScanDB(t)
	result := sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100)

	if err := db.InsertSBOMResult(result, "sha256:img1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	stored, err := db.GetSBOMResultByID(result.ID)
	if err != nil {
		t.Fatalf("GetSBOMResultByID() error = %v", err)
	}
	if stored == nil || len(stored.Components) == 0 {
		t.Fatalf("stored components missing: %+v", stored)
	}
	if stored.Components[0].Name == "" {
		t.Fatalf("expected populated component rows, got %+v", stored.Components)
	}
}

func TestGetSBOMResultByIDReturnsNilWhenNotFound(t *testing.T) {
	db := newTestScanDB(t)

	result, err := db.GetSBOMResultByID("missing")
	if err != nil {
		t.Fatalf("GetSBOMResultByID() error = %v", err)
	}
	if result != nil {
		t.Fatalf("GetSBOMResultByID() = %+v, want nil", result)
	}
}

func TestQuerySBOMHistoryFiltersByImageHostFormat(t *testing.T) {
	db := newTestScanDB(t)

	results := []models.SBOMResult{
		sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100),
		sampleSBOMResult("sbom-2", "alpine:3.18", "remote", models.SBOMFormatSPDX, 200),
		sampleSBOMResult("sbom-3", "redis:7", "local", models.SBOMFormatCycloneDX, 300),
	}
	for i, result := range results {
		if err := db.InsertSBOMResult(result, "sha256:test-"+string(rune('a'+i))); err != nil {
			t.Fatalf("InsertSBOMResult(%s) error = %v", result.ID, err)
		}
	}

	page, err := db.QuerySBOMHistory(SBOMHistoryQuery{
		ImageRef: "alpine",
		Host:     "local",
		Format:   string(models.SBOMFormatSPDX),
	})
	if err != nil {
		t.Fatalf("QuerySBOMHistory() error = %v", err)
	}
	if len(page.Results) != 1 {
		t.Fatalf("filtered results = %d, want 1", len(page.Results))
	}
	if page.Results[0].ID != "sbom-1" {
		t.Fatalf("filtered result ID = %q, want sbom-1", page.Results[0].ID)
	}
}

func TestQuerySBOMHistoryPagination(t *testing.T) {
	db := newTestScanDB(t)

	for i := 1; i <= 3; i++ {
		result := sampleSBOMResult(
			"sbom-"+string(rune('0'+i)),
			"alpine:3.18",
			"local",
			models.SBOMFormatSPDX,
			int64(i*100),
		)
		if err := db.InsertSBOMResult(result, "sha256:test"); err != nil {
			t.Fatalf("InsertSBOMResult(%s) error = %v", result.ID, err)
		}
	}

	page, err := db.QuerySBOMHistory(SBOMHistoryQuery{
		Page:     2,
		PageSize: 1,
		SortBy:   "completed_at",
		SortDir:  "desc",
	})
	if err != nil {
		t.Fatalf("QuerySBOMHistory() error = %v", err)
	}
	if page.Total != 3 || page.TotalPages != 3 {
		t.Fatalf("unexpected pagination metadata: %+v", page)
	}
	if len(page.Results) != 1 {
		t.Fatalf("page results = %d, want 1", len(page.Results))
	}
	if page.Results[0].CompletedAt != 200 {
		t.Fatalf("page result completed_at = %d, want 200", page.Results[0].CompletedAt)
	}
}

func TestListSBOMedImagesDistinctPairs(t *testing.T) {
	db := newTestScanDB(t)

	if err := db.InsertSBOMResult(sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100), "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}
	if err := db.InsertSBOMResult(sampleSBOMResult("sbom-2", "alpine:3.18", "local", models.SBOMFormatCycloneDX, 200), "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}
	if err := db.InsertSBOMResult(sampleSBOMResult("sbom-3", "redis:7", "remote", models.SBOMFormatSPDX, 300), "sha256:2"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	images, err := db.ListSBOMedImages()
	if err != nil {
		t.Fatalf("ListSBOMedImages() error = %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("ListSBOMedImages() count = %d, want 2", len(images))
	}
	if images[0].LastSBOMAt < images[1].LastSBOMAt {
		t.Fatalf("expected images ordered by last_sbom_at desc, got %+v", images)
	}
}

func TestCanRegenerateSBOMNeverGenerated(t *testing.T) {
	db := newTestScanDB(t)

	canRegenerate, err := db.CanRegenerateSBOM("local", "alpine:3.18", "sha256:new")
	if err != nil {
		t.Fatalf("CanRegenerateSBOM() error = %v", err)
	}
	if !canRegenerate {
		t.Fatal("expected regenerate allowed for never-generated image")
	}
}

func TestCanRegenerateSBOMImageChanged(t *testing.T) {
	db := newTestScanDB(t)

	if err := db.UpsertImageSBOMState("local", "alpine:3.18", "sha256:old", 100, "sbom-1"); err != nil {
		t.Fatalf("UpsertImageSBOMState() error = %v", err)
	}

	canRegenerate, err := db.CanRegenerateSBOM("local", "alpine:3.18", "sha256:new")
	if err != nil {
		t.Fatalf("CanRegenerateSBOM() error = %v", err)
	}
	if !canRegenerate {
		t.Fatal("expected regenerate allowed when image digest changed")
	}
}

func TestCanRegenerateSBOMImageUnchanged(t *testing.T) {
	db := newTestScanDB(t)

	if err := db.UpsertImageSBOMState("local", "alpine:3.18", "sha256:same", 100, "sbom-1"); err != nil {
		t.Fatalf("UpsertImageSBOMState() error = %v", err)
	}

	canRegenerate, err := db.CanRegenerateSBOM("local", "alpine:3.18", "sha256:same")
	if err != nil {
		t.Fatalf("CanRegenerateSBOM() error = %v", err)
	}
	if canRegenerate {
		t.Fatal("expected regenerate blocked when image digest unchanged")
	}
}

func TestDeleteSBOMResultCascadeDeletesComponents(t *testing.T) {
	db := newTestScanDB(t)
	result := sampleSBOMResult("sbom-1", "alpine:3.18", "local", models.SBOMFormatSPDX, 100)

	if err := db.InsertSBOMResult(result, "sha256:img1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}
	if err := db.DeleteSBOMResult(result.ID); err != nil {
		t.Fatalf("DeleteSBOMResult() error = %v", err)
	}

	stored, err := db.GetSBOMResultByID(result.ID)
	if err != nil {
		t.Fatalf("GetSBOMResultByID() error = %v", err)
	}
	if stored != nil {
		t.Fatalf("GetSBOMResultByID() after delete = %+v, want nil", stored)
	}

	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM sbom_components WHERE sbom_result_id = ?`, result.ID).Scan(&count); err != nil {
		t.Fatalf("count sbom components error = %v", err)
	}
	if count != 0 {
		t.Fatalf("sbom component count after delete = %d, want 0", count)
	}
}

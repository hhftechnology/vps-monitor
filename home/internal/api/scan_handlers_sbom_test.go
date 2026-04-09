package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
)

func testSBOMResult(id string, completedAt int64, filePath string) models.SBOMResult {
	return models.SBOMResult{
		ID:             id,
		ImageRef:       "alpine:3.18",
		Host:           "local",
		Format:         models.SBOMFormatSPDX,
		ComponentCount: 2,
		FileSize:       128,
		FilePath:       filePath,
		StartedAt:      completedAt - 5,
		CompletedAt:    completedAt,
		DurationMs:     5000,
		Components: []models.SBOMComponent{
			{Name: "busybox", Version: "1.36.1-r7", Type: "package", PURL: "pkg:apk/alpine/busybox@1.36.1-r7"},
			{Name: "ssl_client", Version: "1.36.1-r7", Type: "package", PURL: "pkg:apk/alpine/ssl_client@1.36.1-r7"},
		},
	}
}

func TestStartSBOMGeneration409WhenImageUnchanged(t *testing.T) {
	svc := newTestScannerService(t)
	if err := svc.Store().DB().UpsertImageSBOMState("local", "alpine:3.18", string(models.SBOMFormatSPDX), "sha256:same", 100, "sbom-1"); err != nil {
		t.Fatalf("UpsertImageSBOMState() error = %v", err)
	}

	h := &ScanHandlers{
		scanner:          svc,
		resolveImageIDFn: func(host, imageRef string) string { return "sha256:same" },
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/sbom", bytes.NewBufferString(`{"imageRef":"alpine:3.18","host":"local","format":"spdx-json"}`))
	rec := httptest.NewRecorder()
	h.StartSBOMGeneration(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["error"] != "image_unchanged" {
		t.Fatalf("expected image_unchanged error, got %+v", body)
	}
	if _, ok := body["last_sbom_id"]; !ok {
		t.Fatalf("expected last_sbom_id in response, got %+v", body)
	}
}

func TestStartSBOMGenerationBypassesGateWhenForceTrue(t *testing.T) {
	svc := newTestScannerService(t)
	if err := svc.Store().DB().UpsertImageSBOMState("local", "alpine:3.18", string(models.SBOMFormatSPDX), "sha256:same", 100, "sbom-1"); err != nil {
		t.Fatalf("UpsertImageSBOMState() error = %v", err)
	}

	h := &ScanHandlers{
		scanner:          svc,
		resolveImageIDFn: func(host, imageRef string) string { return "sha256:same" },
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/sbom", bytes.NewBufferString(`{"imageRef":"alpine:3.18","host":"local","format":"spdx-json","force":true}`))
	rec := httptest.NewRecorder()
	h.StartSBOMGeneration(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestGetSBOMHistoryReturnsPagedResults(t *testing.T) {
	svc := newTestScannerService(t)
	h := &ScanHandlers{scanner: svc}

	filePath1 := filepath.Join(t.TempDir(), "sbom-1.json")
	filePath2 := filepath.Join(t.TempDir(), "sbom-2.json")
	if err := svc.Store().DB().InsertSBOMResult(testSBOMResult("sbom-1", 100, filePath1), "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult(sbom-1) error = %v", err)
	}
	if err := svc.Store().DB().InsertSBOMResult(testSBOMResult("sbom-2", 200, filePath2), "sha256:2"); err != nil {
		t.Fatalf("InsertSBOMResult(sbom-2) error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/sbom/history?page=1&page_size=1", nil)
	rec := httptest.NewRecorder()
	h.GetSBOMHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var page scanner.SBOMHistoryPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if page.Total != 2 || page.PageSize != 1 || len(page.Results) != 1 {
		t.Fatalf("unexpected page payload: %+v", page)
	}
	if page.Results[0].ID != "sbom-2" {
		t.Fatalf("expected newest result first, got %+v", page.Results[0])
	}
}

func TestDeleteSBOMHistoryRemovesRowAndFile(t *testing.T) {
	svc := newTestScannerService(t)
	filePath := filepath.Join(t.TempDir(), "sbom-delete.json")
	if err := os.WriteFile(filePath, []byte(`{"bomFormat":"CycloneDX"}`), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	result := testSBOMResult("sbom-delete", 100, filePath)
	if err := svc.Store().DB().InsertSBOMResult(result, "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scan/sbom/history/sbom-delete", nil)
	req = chiContext(req, map[string]string{"id": "sbom-delete"})
	rec := httptest.NewRecorder()
	h.DeleteSBOMHistory(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (%s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact removed, stat err = %v", err)
	}
	stored, err := svc.Store().DB().GetSBOMResultByID("sbom-delete")
	if err != nil {
		t.Fatalf("GetSBOMResultByID() error = %v", err)
	}
	if stored != nil {
		t.Fatalf("expected deleted row, got %+v", stored)
	}
}

func TestDownloadSBOMHistoryReturnsJSON(t *testing.T) {
	svc := newTestScannerService(t)
	filePath := filepath.Join(t.TempDir(), "sbom-download.json")
	content := []byte(`{"bomFormat":"CycloneDX","components":[{"name":"busybox"}]}`)
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := svc.Store().DB().InsertSBOMResult(testSBOMResult("sbom-download", 100, filePath), "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/sbom/history/sbom-download/download", nil)
	req = chiContext(req, map[string]string{"id": "sbom-download"})
	rec := httptest.NewRecorder()
	h.DownloadSBOMHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != string(content) {
		t.Fatalf("download body mismatch: got %q want %q", got, string(content))
	}
}

func TestDownloadSBOMHistory410WhenFileMissing(t *testing.T) {
	svc := newTestScannerService(t)
	filePath := filepath.Join(t.TempDir(), "missing.json")

	if err := svc.Store().DB().InsertSBOMResult(testSBOMResult("sbom-missing", 100, filePath), "sha256:1"); err != nil {
		t.Fatalf("InsertSBOMResult() error = %v", err)
	}

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/sbom/history/sbom-missing/download", nil)
	req = chiContext(req, map[string]string{"id": "sbom-missing"})
	rec := httptest.NewRecorder()
	h.DownloadSBOMHistory(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}
}

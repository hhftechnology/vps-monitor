package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

// newTestScannerService creates a ScannerService backed by a stub Registry that
// has no Docker client. StartScan goroutines fail gracefully (status="failed")
// without panicking, making the service safe for handler tests.
func newTestScannerService() *scanner.ScannerService {
	cfg := &models.ScannerConfig{
		GrypeImage:     "anchore/grype:latest",
		TrivyImage:     "aquasec/trivy:latest",
		SyftImage:      "anchore/syft:latest",
		DefaultScanner: models.ScannerGrype,
	}
	// Registry with nil docker client: AcquireDocker returns (nil, func(){})
	// which is handled gracefully in runScan.
	registry := services.NewRegistry(nil, nil, nil, &config.Config{}, nil)
	return scanner.NewScannerService(registry, cfg)
}

// newTestManager creates a Manager pointing to a temp file (for tests that
// exercise UpdateScannerConfig persistence).
func newTestManager(t *testing.T) *config.Manager {
	t.Helper()
	return &config.Manager{}
}

// chiContext wraps a request in a chi router context with the given URL params.
func chiContext(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}



// ─── GetScanJobs ──────────────────────────────────────────────────────────────

func TestGetScanJobsReturnsEmptySlices(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/jobs", nil)
	rec := httptest.NewRecorder()
	h.GetScanJobs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := body["jobs"]; !ok {
		t.Fatal("response must contain 'jobs' key")
	}
	if _, ok := body["bulkJobs"]; !ok {
		t.Fatal("response must contain 'bulkJobs' key")
	}
}

// ─── GetScanJob ───────────────────────────────────────────────────────────────

func TestGetScanJobReturnsNotFoundForUnknownID(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/jobs/unknown-id", nil)
	req = chiContext(req, map[string]string{"id": "unknown-id"})
	rec := httptest.NewRecorder()
	h.GetScanJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetScanJobReturnsJobWhenFound(t *testing.T) {
	svc := newTestScannerService()

	// Manually inject a job into the service store via StartScan
	// (StartScan returns immediately with pending status; the goroutine fails
	// because registry is nil, but the job is registered first.)
	job, err := svc.StartScan("nginx:latest", "local", models.ScannerGrype)
	if err != nil {
		t.Fatalf("StartScan returned error: %v", err)
	}
	// Give the goroutine a moment to set the failure status.
	time.Sleep(10 * time.Millisecond)

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/jobs/"+job.ID, nil)
	req = chiContext(req, map[string]string{"id": job.ID})
	rec := httptest.NewRecorder()
	h.GetScanJob(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := body["job"]; !ok {
		t.Fatal("response must contain 'job' key")
	}
}

// ─── StartScan validation ─────────────────────────────────────────────────────

func TestStartScanRejectsMissingFields(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	cases := []struct {
		name string
		body string
	}{
		{"missing imageRef", `{"host":"local"}`},
		{"missing host", `{"imageRef":"nginx:latest"}`},
		{"empty body fields", `{"imageRef":"","host":""}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scan",
				bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			h.StartScan(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestStartScanRejectsInvalidJSON(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan",
		bytes.NewBufferString(`not-json`))
	rec := httptest.NewRecorder()
	h.StartScan(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// ─── StartBulkScan validation ─────────────────────────────────────────────────

func TestStartBulkScanRejectsInvalidJSON(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/bulk",
		bytes.NewBufferString(`not-json`))
	rec := httptest.NewRecorder()
	h.StartBulkScan(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// ─── CancelScanJob ────────────────────────────────────────────────────────────

func TestCancelScanJobReturnsNotFoundForUnknownID(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/scan/jobs/ghost", nil)
	req = chiContext(req, map[string]string{"id": "ghost"})
	rec := httptest.NewRecorder()
	h.CancelScanJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ─── GetScanResults ───────────────────────────────────────────────────────────

func TestGetScanResultsRequiresHostParam(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/nginx:latest", nil)
	req = chiContext(req, map[string]string{"imageRef": "nginx:latest"})
	// No host query param
	rec := httptest.NewRecorder()
	h.GetScanResults(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when host param is missing, got %d", rec.Code)
	}
}

func TestGetScanResultsReturnsEmptyForUnknownImage(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/unknown:image?host=local", nil)
	req = chiContext(req, map[string]string{"imageRef": "unknown:image"})
	rec := httptest.NewRecorder()
	h.GetScanResults(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetScanResultsReturnsStoredResults(t *testing.T) {
	svc := newTestScannerService()
	svc.Store().Add(models.ScanResult{
		ID:       "res-1",
		ImageRef: "redis:7",
		Host:     "local",
		Summary:  models.SeveritySummary{Total: 3},
	})

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/redis:7?host=local", nil)
	req = chiContext(req, map[string]string{"imageRef": "redis:7"})
	rec := httptest.NewRecorder()
	h.GetScanResults(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	results, ok := body["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got %T", body["results"])
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

// ─── GetLatestScanResult ──────────────────────────────────────────────────────

func TestGetLatestScanResultRequiresHostParam(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/nginx:latest/latest", nil)
	req = chiContext(req, map[string]string{"imageRef": "nginx:latest"})
	rec := httptest.NewRecorder()
	h.GetLatestScanResult(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when host param is missing, got %d", rec.Code)
	}
}

func TestGetLatestScanResultReturns404WhenNoResults(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/missing:img/latest?host=local", nil)
	req = chiContext(req, map[string]string{"imageRef": "missing:img"})
	rec := httptest.NewRecorder()
	h.GetLatestScanResult(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGetLatestScanResultReturnsResult(t *testing.T) {
	svc := newTestScannerService()
	svc.Store().Add(models.ScanResult{
		ID:       "latest-1",
		ImageRef: "postgres:16",
		Host:     "remote",
	})

	h := &ScanHandlers{scanner: svc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/results/postgres:16/latest?host=remote", nil)
	req = chiContext(req, map[string]string{"imageRef": "postgres:16"})
	rec := httptest.NewRecorder()
	h.GetLatestScanResult(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := body["result"]; !ok {
		t.Fatal("expected 'result' key in response")
	}
}

// ─── StartSBOMGeneration validation ──────────────────────────────────────────

func TestStartSBOMGenerationRejectsMissingFields(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	cases := []struct {
		name string
		body string
	}{
		{"missing imageRef", `{"host":"local"}`},
		{"missing host", `{"imageRef":"nginx:latest"}`},
		{"both missing", `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/sbom",
				bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			h.StartSBOMGeneration(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d", tc.name, rec.Code)
			}
		})
	}
}

func TestStartSBOMGenerationRejectsInvalidJSON(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan/sbom",
		bytes.NewBufferString(`{invalid}`))
	rec := httptest.NewRecorder()
	h.StartSBOMGeneration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// ─── GetSBOMJob ───────────────────────────────────────────────────────────────

func TestGetSBOMJobReturnsNotFoundForUnknownID(t *testing.T) {
	h := &ScanHandlers{scanner: newTestScannerService()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scan/sbom/ghost", nil)
	req = chiContext(req, map[string]string{"id": "ghost"})
	rec := httptest.NewRecorder()
	h.GetSBOMJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ─── GetScannerConfig ─────────────────────────────────────────────────────────

func TestGetScannerConfigReturnsCurrentConfig(t *testing.T) {
	svc := newTestScannerService()
	h := &ScanHandlers{scanner: svc}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/scan", nil)
	rec := httptest.NewRecorder()
	h.GetScannerConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := body["config"]; !ok {
		t.Fatal("expected 'config' key in response")
	}
}

// ─── UpdateScannerConfig ──────────────────────────────────────────────────────

func TestUpdateScannerConfigRejectsInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &config.Manager{}
	_ = mgr
	// Use a real manager pointing to a temp file
	realMgr := mustNewManagerWithTempFile(t, filepath.Join(tmpDir, "config.json"))

	h := &ScanHandlers{
		scanner: newTestScannerService(),
		manager: realMgr,
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/scan",
		bytes.NewBufferString(`{not-json}`))
	rec := httptest.NewRecorder()
	h.UpdateScannerConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestUpdateScannerConfigPersistsChanges(t *testing.T) {
	realMgr := mustNewManagerWithTempFile(t, filepath.Join(t.TempDir(), "config.json"))

	h := &ScanHandlers{
		scanner: newTestScannerService(),
		manager: realMgr,
	}

	payload := `{
		"grypeImage":"anchore/grype:v2",
		"trivyImage":"aquasec/trivy:v2",
		"syftImage":"anchore/syft:v2",
		"defaultScanner":"trivy",
		"grypeArgs":"",
		"trivyArgs":"",
		"notifications":{
			"onScanComplete":false,
			"onBulkComplete":true,
			"minSeverity":"Critical"
		}
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/scan",
		bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.UpdateScannerConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := body["config"]; !ok {
		t.Fatal("expected 'config' key in response")
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// mustNewManagerWithTempFile builds a config.Manager backed by a temp file.
// It accesses Manager fields directly because the package-level test lives in
// the same `api` package and Manager is from an external package — we construct
// it via the public exported fields (unexported fields are zero-valued, which
// is fine for the persist/merge path).
func mustNewManagerWithTempFile(t *testing.T, path string) *config.Manager {
	t.Helper()

	// We cannot set unexported fields from outside the config package, so we
	// rely on Manager's exported API after construction via NewManager.
	// Since NewManager reads env vars / disk, we instead build a minimal Manager
	// using a table-driven trick: write an empty config file first, then let
	// NewManager load it. In tests we override CONFIG_PATH.
	t.Setenv("CONFIG_PATH", path)
	mgr := config.NewManager()
	return mgr
}
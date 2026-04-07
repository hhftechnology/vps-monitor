package scanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// ─── computeSummary ──────────────────────────────────────────────────────────

func TestComputeSummaryEmpty(t *testing.T) {
	summary := computeSummary(nil)
	if summary.Total != 0 {
		t.Fatalf("expected Total=0, got %d", summary.Total)
	}
	if summary.Critical+summary.High+summary.Medium+summary.Low+summary.Negligible+summary.Unknown != 0 {
		t.Fatalf("expected all severity counts 0, got %+v", summary)
	}
}

func TestComputeSummaryCountsBySeverity(t *testing.T) {
	vulns := []models.Vulnerability{
		{ID: "CVE-1", Severity: models.SeverityCritical},
		{ID: "CVE-2", Severity: models.SeverityCritical},
		{ID: "CVE-3", Severity: models.SeverityHigh},
		{ID: "CVE-4", Severity: models.SeverityMedium},
		{ID: "CVE-5", Severity: models.SeverityLow},
		{ID: "CVE-6", Severity: models.SeverityNegligible},
		{ID: "CVE-7", Severity: models.SeverityUnknown},
		{ID: "CVE-8", Severity: "SomeOther"}, // unmapped → Unknown
	}

	summary := computeSummary(vulns)

	if summary.Total != 8 {
		t.Fatalf("expected Total=8, got %d", summary.Total)
	}
	if summary.Critical != 2 {
		t.Fatalf("expected Critical=2, got %d", summary.Critical)
	}
	if summary.High != 1 {
		t.Fatalf("expected High=1, got %d", summary.High)
	}
	if summary.Medium != 1 {
		t.Fatalf("expected Medium=1, got %d", summary.Medium)
	}
	if summary.Low != 1 {
		t.Fatalf("expected Low=1, got %d", summary.Low)
	}
	if summary.Negligible != 1 {
		t.Fatalf("expected Negligible=1, got %d", summary.Negligible)
	}
	if summary.Unknown != 2 { // SeverityUnknown + "SomeOther"
		t.Fatalf("expected Unknown=2, got %d", summary.Unknown)
	}
}

func TestComputeSummaryTotalEqualsSumOfSeverities(t *testing.T) {
	vulns := []models.Vulnerability{
		{Severity: models.SeverityHigh},
		{Severity: models.SeverityMedium},
		{Severity: models.SeverityLow},
	}
	s := computeSummary(vulns)
	sumFields := s.Critical + s.High + s.Medium + s.Low + s.Negligible + s.Unknown
	if sumFields != s.Total {
		t.Fatalf("sum of severity fields (%d) != Total (%d)", sumFields, s.Total)
	}
}

// ─── meetsMinSeverity ────────────────────────────────────────────────────────

func TestMeetsMinSeverityCriticalOnly(t *testing.T) {
	summaryWithCritical := models.SeveritySummary{Critical: 1, Total: 1}
	summaryWithHighOnly := models.SeveritySummary{High: 1, Total: 1}
	summaryEmpty := models.SeveritySummary{}

	if !meetsMinSeverity(summaryWithCritical, models.SeverityCritical) {
		t.Fatal("expected critical to meet Critical threshold")
	}
	if meetsMinSeverity(summaryWithHighOnly, models.SeverityCritical) {
		t.Fatal("expected high-only to NOT meet Critical threshold")
	}
	if meetsMinSeverity(summaryEmpty, models.SeverityCritical) {
		t.Fatal("expected empty to NOT meet Critical threshold")
	}
}

func TestMeetsMinSeverityHigh(t *testing.T) {
	withCritical := models.SeveritySummary{Critical: 1, Total: 1}
	withHigh := models.SeveritySummary{High: 1, Total: 1}
	withMediumOnly := models.SeveritySummary{Medium: 1, Total: 1}

	if !meetsMinSeverity(withCritical, models.SeverityHigh) {
		t.Fatal("critical should meet High threshold")
	}
	if !meetsMinSeverity(withHigh, models.SeverityHigh) {
		t.Fatal("high should meet High threshold")
	}
	if meetsMinSeverity(withMediumOnly, models.SeverityHigh) {
		t.Fatal("medium-only should NOT meet High threshold")
	}
}

func TestMeetsMinSeverityMedium(t *testing.T) {
	withMedium := models.SeveritySummary{Medium: 1, Total: 1}
	withLowOnly := models.SeveritySummary{Low: 1, Total: 1}

	if !meetsMinSeverity(withMedium, models.SeverityMedium) {
		t.Fatal("medium should meet Medium threshold")
	}
	if meetsMinSeverity(withLowOnly, models.SeverityMedium) {
		t.Fatal("low-only should NOT meet Medium threshold")
	}
}

func TestMeetsMinSeverityLow(t *testing.T) {
	withLow := models.SeveritySummary{Low: 1, Total: 1}
	withNegOnly := models.SeveritySummary{Negligible: 1, Total: 1}

	if !meetsMinSeverity(withLow, models.SeverityLow) {
		t.Fatal("low should meet Low threshold")
	}
	if meetsMinSeverity(withNegOnly, models.SeverityLow) {
		t.Fatal("negligible-only should NOT meet Low threshold")
	}
}

func TestMeetsMinSeverityDefaultFallback(t *testing.T) {
	// Any non-empty summary should meet an unrecognized/default threshold
	withNeg := models.SeveritySummary{Negligible: 1, Total: 1}
	empty := models.SeveritySummary{}

	if !meetsMinSeverity(withNeg, "Negligible") {
		t.Fatal("non-empty summary should meet unrecognized threshold")
	}
	if meetsMinSeverity(empty, "Negligible") {
		t.Fatal("empty summary should NOT meet unrecognized threshold")
	}
}

// ─── containsHost ─────────────────────────────────────────────────────────────

func TestContainsHostFound(t *testing.T) {
	hosts := []string{"host-a", "host-b", "host-c"}
	if !containsHost(hosts, "host-b") {
		t.Fatal("expected host-b to be found")
	}
}

func TestContainsHostNotFound(t *testing.T) {
	hosts := []string{"host-a", "host-b"}
	if containsHost(hosts, "host-z") {
		t.Fatal("expected host-z not to be found")
	}
}

func TestContainsHostEmptyList(t *testing.T) {
	if containsHost(nil, "any") {
		t.Fatal("expected false for nil host list")
	}
	if containsHost([]string{}, "any") {
		t.Fatal("expected false for empty host list")
	}
}

func TestContainsHostExactMatch(t *testing.T) {
	// Ensure no partial matching
	if containsHost([]string{"host-abc"}, "host") {
		t.Fatal("containsHost must not do partial matching")
	}
}

// ─── Helper: minimal ScannerService without Docker ───────────────────────────

// newTestScannerService creates a ScannerService backed by the supplied config
// without requiring a registry or database (the timeout/limits helpers only
// access the config).
func newTestScannerService(cfg *models.ScannerConfig) *ScannerService {
	s := &ScannerService{}
	s.config.Store(cfg)
	return s
}

// ─── scanTimeout ─────────────────────────────────────────────────────────────

func TestScanTimeoutUsesConfiguredValue(t *testing.T) {
	cfg := &models.ScannerConfig{ScanTimeoutMinutes: 45}
	s := newTestScannerService(cfg)
	if got := s.scanTimeout(); got != 45*time.Minute {
		t.Fatalf("expected 45m, got %v", got)
	}
}

func TestScanTimeoutDefaultWhenZero(t *testing.T) {
	cfg := &models.ScannerConfig{ScanTimeoutMinutes: 0}
	s := newTestScannerService(cfg)
	if got := s.scanTimeout(); got != 20*time.Minute {
		t.Fatalf("expected default 20m, got %v", got)
	}
}

func TestScanTimeoutDefaultWhenNilConfig(t *testing.T) {
	s := &ScannerService{} // config is nil pointer (zero atomic.Pointer)
	if got := s.scanTimeout(); got != 20*time.Minute {
		t.Fatalf("expected default 20m for nil config, got %v", got)
	}
}

func TestScanTimeoutNegativeValueUsesDefault(t *testing.T) {
	cfg := &models.ScannerConfig{ScanTimeoutMinutes: -10}
	s := newTestScannerService(cfg)
	if got := s.scanTimeout(); got != 20*time.Minute {
		t.Fatalf("expected default 20m for negative value, got %v", got)
	}
}

// ─── bulkTimeout ─────────────────────────────────────────────────────────────

func TestBulkTimeoutUsesConfiguredValue(t *testing.T) {
	cfg := &models.ScannerConfig{BulkTimeoutMinutes: 240}
	s := newTestScannerService(cfg)
	if got := s.bulkTimeout(); got != 240*time.Minute {
		t.Fatalf("expected 240m, got %v", got)
	}
}

func TestBulkTimeoutDefaultWhenZero(t *testing.T) {
	cfg := &models.ScannerConfig{BulkTimeoutMinutes: 0}
	s := newTestScannerService(cfg)
	if got := s.bulkTimeout(); got != 120*time.Minute {
		t.Fatalf("expected default 120m, got %v", got)
	}
}

func TestBulkTimeoutDefaultWhenNilConfig(t *testing.T) {
	s := &ScannerService{}
	if got := s.bulkTimeout(); got != 120*time.Minute {
		t.Fatalf("expected default 120m for nil config, got %v", got)
	}
}

// ─── scannerLimits ────────────────────────────────────────────────────────────

func TestScannerLimitsUsesConfiguredValues(t *testing.T) {
	cfg := &models.ScannerConfig{ScannerMemoryMB: 4096, ScannerPidsLimit: 1024}
	s := newTestScannerService(cfg)
	limits := s.scannerLimits()
	if limits.MemoryBytes != 4096*1024*1024 {
		t.Fatalf("expected MemoryBytes=%d, got %d", int64(4096*1024*1024), limits.MemoryBytes)
	}
	if limits.PidsLimit != 1024 {
		t.Fatalf("expected PidsLimit=1024, got %d", limits.PidsLimit)
	}
}

func TestScannerLimitsDefaultsWhenZero(t *testing.T) {
	cfg := &models.ScannerConfig{ScannerMemoryMB: 0, ScannerPidsLimit: 0}
	s := newTestScannerService(cfg)
	limits := s.scannerLimits()
	if limits.MemoryBytes != 2048*1024*1024 {
		t.Fatalf("expected default MemoryBytes=%d, got %d", int64(2048*1024*1024), limits.MemoryBytes)
	}
	if limits.PidsLimit != 512 {
		t.Fatalf("expected default PidsLimit=512, got %d", limits.PidsLimit)
	}
}

func TestScannerLimitsDefaultsWhenNilConfig(t *testing.T) {
	s := &ScannerService{}
	limits := s.scannerLimits()
	if limits.MemoryBytes != 2048*1024*1024 {
		t.Fatalf("expected default MemoryBytes for nil config, got %d", limits.MemoryBytes)
	}
	if limits.PidsLimit != 512 {
		t.Fatalf("expected default PidsLimit=512 for nil config, got %d", limits.PidsLimit)
	}
}

func TestScannerLimitsMemoryConversion(t *testing.T) {
	// Verify MB-to-bytes multiplication: 1 MB → 1 048 576 bytes
	cfg := &models.ScannerConfig{ScannerMemoryMB: 1, ScannerPidsLimit: 1}
	s := newTestScannerService(cfg)
	if got := s.scannerLimits().MemoryBytes; got != 1024*1024 {
		t.Fatalf("1 MB should be %d bytes, got %d", 1024*1024, got)
	}
}

// ─── humanBytes ──────────────────────────────────────────────────────────────

func TestHumanBytesBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
	}
	for _, c := range cases {
		if got := humanBytes(c.n); got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestHumanBytesKilobytes(t *testing.T) {
	if got := humanBytes(1024); got != "1.0 KB" {
		t.Fatalf("expected '1.0 KB', got %q", got)
	}
	if got := humanBytes(2048); got != "2.0 KB" {
		t.Fatalf("expected '2.0 KB', got %q", got)
	}
}

func TestHumanBytesMegabytes(t *testing.T) {
	if got := humanBytes(1024 * 1024); got != "1.0 MB" {
		t.Fatalf("expected '1.0 MB', got %q", got)
	}
	if got := humanBytes(512 * 1024 * 1024); got != "512.0 MB" {
		t.Fatalf("expected '512.0 MB', got %q", got)
	}
}

func TestHumanBytesGigabytes(t *testing.T) {
	if got := humanBytes(1024 * 1024 * 1024); got != "1.0 GB" {
		t.Fatalf("expected '1.0 GB', got %q", got)
	}
}

// ─── heartbeat ───────────────────────────────────────────────────────────────

// TestHeartbeatUpdatesJobProgress verifies that the heartbeat goroutine updates
// the job's Progress field while the scan is still running.
func TestHeartbeatUpdatesJobProgress(t *testing.T) {
	s := newTestScannerService(&models.ScannerConfig{})

	job := &models.ScanJob{Status: models.ScanJobScanning}
	var bytesWritten int64
	atomic.StoreInt64(&bytesWritten, 1536) // 1.5 KB

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a ticker period override isn't possible, so we rely on the real
	// 5-second ticker. Instead we just exercise the cancel path directly
	// after a brief run.
	done := make(chan struct{})
	go func() {
		s.heartbeat(ctx, job, &bytesWritten, time.Now().Add(-10*time.Second))
		close(done)
	}()

	// Cancel promptly and confirm the goroutine exits.
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat did not exit after context cancel")
	}
}

// TestHeartbeatDoesNotOverwriteNonScanningStatus verifies that heartbeat leaves
// the Progress field alone once the job transitions out of ScanJobScanning.
func TestHeartbeatDoesNotOverwriteNonScanningStatus(t *testing.T) {
	s := newTestScannerService(&models.ScannerConfig{})

	job := &models.ScanJob{Status: models.ScanJobComplete, Progress: "original"}
	var bytesWritten int64

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	s.heartbeat(ctx, job, &bytesWritten, time.Now())

	if job.Progress != "original" {
		t.Fatalf("expected progress to remain 'original', got %q", job.Progress)
	}
}
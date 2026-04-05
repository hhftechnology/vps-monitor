package scanner

import (
	"testing"

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
package scanner

import (
	"fmt"
	"strings"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// AnomalyDiff represents the diff between two scans, highlighting new vulnerabilities.
type AnomalyDiff struct {
	NewVulnerabilities []models.Vulnerability `json:"new_vulnerabilities"`
	Summary            models.SeveritySummary `json:"summary"`
	Message            string                 `json:"message"`
}

// findNewVulnerabilities returns vulnerabilities present in current but not in previous.
// Comparison key: vulnerability ID + package name.
func findNewVulnerabilities(previous, current []models.Vulnerability) []models.Vulnerability {
	prevSet := make(map[string]struct{}, len(previous))
	for _, v := range previous {
		prevSet[vulnKey(v)] = struct{}{}
	}

	var newVulns []models.Vulnerability
	for _, v := range current {
		if _, exists := prevSet[vulnKey(v)]; !exists {
			newVulns = append(newVulns, v)
		}
	}
	return newVulns
}

// filterBySeverity filters vulnerabilities to only those meeting the minimum severity.
func filterBySeverity(vulns []models.Vulnerability, minSeverity models.SeverityLevel) []models.Vulnerability {
	minLevel := severityLevel(minSeverity)
	if minLevel == 0 {
		return vulns // no filter or unknown
	}

	var filtered []models.Vulnerability
	for _, v := range vulns {
		if severityLevel(v.Severity) >= minLevel {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// computeAnomalyDiff builds the anomaly diff summary from new vulnerabilities.
func computeAnomalyDiff(newVulns []models.Vulnerability) AnomalyDiff {
	diff := AnomalyDiff{
		NewVulnerabilities: newVulns,
	}

	for _, v := range newVulns {
		diff.Summary.Total++
		switch v.Severity {
		case models.SeverityCritical:
			diff.Summary.Critical++
		case models.SeverityHigh:
			diff.Summary.High++
		case models.SeverityMedium:
			diff.Summary.Medium++
		case models.SeverityLow:
			diff.Summary.Low++
		case models.SeverityNegligible:
			diff.Summary.Negligible++
		default:
			diff.Summary.Unknown++
		}
	}

	// Build human-readable message
	var parts []string
	if diff.Summary.Critical > 0 {
		parts = append(parts, fmt.Sprintf("%d Critical", diff.Summary.Critical))
	}
	if diff.Summary.High > 0 {
		parts = append(parts, fmt.Sprintf("%d High", diff.Summary.High))
	}
	if diff.Summary.Medium > 0 {
		parts = append(parts, fmt.Sprintf("%d Medium", diff.Summary.Medium))
	}
	if diff.Summary.Low > 0 {
		parts = append(parts, fmt.Sprintf("%d Low", diff.Summary.Low))
	}
	if diff.Summary.Negligible > 0 {
		parts = append(parts, fmt.Sprintf("%d Negligible", diff.Summary.Negligible))
	}
	if diff.Summary.Unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d Unknown", diff.Summary.Unknown))
	}

	if len(parts) > 0 {
		diff.Message = fmt.Sprintf("%d new vulnerabilities found (%s)", diff.Summary.Total, strings.Join(parts, ", "))
	} else {
		diff.Message = fmt.Sprintf("%d new vulnerabilities found", diff.Summary.Total)
	}

	return diff
}

func vulnKey(v models.Vulnerability) string {
	return v.ID + ":" + v.Package
}

func severityLevel(s models.SeverityLevel) int {
	switch s {
	case models.SeverityCritical:
		return 5
	case models.SeverityHigh:
		return 4
	case models.SeverityMedium:
		return 3
	case models.SeverityLow:
		return 2
	case models.SeverityNegligible:
		return 1
	default:
		return 0
	}
}

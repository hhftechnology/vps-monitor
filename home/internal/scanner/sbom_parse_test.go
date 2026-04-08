package scanner

import (
	"path/filepath"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func TestParseSBOMComponentsSPDX(t *testing.T) {
	components, err := parseSBOMComponents(filepath.Join("testdata", "sbom-spdx.json"), models.SBOMFormatSPDX)
	if err != nil {
		t.Fatalf("parseSBOMComponents() error = %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("component count = %d, want 2", len(components))
	}
	if components[0].Type != "package" {
		t.Fatalf("component type = %q, want package", components[0].Type)
	}
	if components[0].PURL == "" {
		t.Fatalf("expected purl for component, got %+v", components[0])
	}
}

func TestParseSBOMComponentsCycloneDX(t *testing.T) {
	components, err := parseSBOMComponents(filepath.Join("testdata", "sbom-cyclonedx.json"), models.SBOMFormatCycloneDX)
	if err != nil {
		t.Fatalf("parseSBOMComponents() error = %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("component count = %d, want 2", len(components))
	}
	if components[0].Type != "library" {
		t.Fatalf("component type = %q, want library", components[0].Type)
	}
	if components[1].Name != "busybox" {
		t.Fatalf("second component name = %q, want busybox", components[1].Name)
	}
}

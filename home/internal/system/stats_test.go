package system

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestHostInfoCPUFieldsExist verifies that HostInfo exposes the two new CPU
// count fields added in this PR and that they are exported correctly.
func TestHostInfoCPUFieldsExist(t *testing.T) {
	info := HostInfo{
		CPULogical:  4,
		CPUPhysical: 2,
	}

	if info.CPULogical != 4 {
		t.Fatalf("expected CPULogical=4, got %d", info.CPULogical)
	}
	if info.CPUPhysical != 2 {
		t.Fatalf("expected CPUPhysical=2, got %d", info.CPUPhysical)
	}
}

// TestHostInfoCPULogicalAlwaysSerialised ensures CPULogical (no omitempty) is
// present in JSON even when zero.
func TestHostInfoCPULogicalAlwaysSerialised(t *testing.T) {
	info := HostInfo{CPULogical: 0, CPUPhysical: 0}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"cpuLogical"`) {
		t.Fatalf("expected cpuLogical key in JSON even when zero, got: %s", jsonStr)
	}
}

// TestHostInfoCPUPhysicalOmittedWhenZero ensures CPUPhysical (has omitempty)
// is absent from JSON when its value is zero.
func TestHostInfoCPUPhysicalOmittedWhenZero(t *testing.T) {
	info := HostInfo{CPULogical: 2, CPUPhysical: 0}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jsonStr := string(data)
	if strings.Contains(jsonStr, `"cpuPhysical"`) {
		t.Fatalf("expected cpuPhysical to be omitted when zero (omitempty), got: %s", jsonStr)
	}
}

// TestHostInfoCPUPhysicalPresentWhenNonZero ensures CPUPhysical is serialised
// when non-zero.
func TestHostInfoCPUPhysicalPresentWhenNonZero(t *testing.T) {
	info := HostInfo{CPULogical: 4, CPUPhysical: 2}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"cpuPhysical":2`) {
		t.Fatalf("expected cpuPhysical:2 in JSON, got: %s", jsonStr)
	}
}

// TestHostInfoCPURoundTrip verifies that HostInfo CPU fields survive a
// marshal/unmarshal round-trip.
func TestHostInfoCPURoundTrip(t *testing.T) {
	original := HostInfo{
		CPULogical:  8,
		CPUPhysical: 4,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var recovered HostInfo
	if err := json.Unmarshal(data, &recovered); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if recovered.CPULogical != original.CPULogical {
		t.Fatalf("CPULogical: expected %d, got %d", original.CPULogical, recovered.CPULogical)
	}
	if recovered.CPUPhysical != original.CPUPhysical {
		t.Fatalf("CPUPhysical: expected %d, got %d", original.CPUPhysical, recovered.CPUPhysical)
	}
}
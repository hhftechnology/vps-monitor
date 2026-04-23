package bot

import (
	"strings"
	"testing"
)

func TestAppendHistoryAveragesIncludesOnlyAvailableWindows(t *testing.T) {
	line := appendHistoryAverages("container", 0, 0, false, 12, 34, true)
	if strings.Contains(line, "1h") {
		t.Fatalf("did not expect 1h segment, got %q", line)
	}
	if !strings.Contains(line, "12h 12.0/34.0") {
		t.Fatalf("expected 12h segment, got %q", line)
	}

	line = appendHistoryAverages("container", 10, 20, true, 0, 0, false)
	if !strings.Contains(line, "1h 10.0/20.0") {
		t.Fatalf("expected 1h segment, got %q", line)
	}
	if strings.Contains(line, "12h") {
		t.Fatalf("did not expect 12h segment, got %q", line)
	}
}

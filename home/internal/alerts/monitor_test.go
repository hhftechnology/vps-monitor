package alerts

import (
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

func TestIsCriticalAlertMatchesThresholdAlertsOnly(t *testing.T) {
	if !isCriticalAlert(models.Alert{Type: models.AlertCPUThreshold}) {
		t.Fatal("expected CPU threshold alerts to be critical")
	}
	if !isCriticalAlert(models.Alert{Type: models.AlertMemoryThreshold}) {
		t.Fatal("expected memory threshold alerts to be critical")
	}
	if isCriticalAlert(models.Alert{Type: models.AlertContainerStopped}) {
		t.Fatal("expected container stopped alerts to be excluded from critical-only filtering")
	}
}

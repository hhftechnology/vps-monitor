package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// AlertHandlers holds dependencies for alert-related handlers
type AlertHandlers struct {
	monitor *alerts.Monitor
	config  *models.AlertConfigResponse
}

// NewAlertHandlers creates new alert handlers
func NewAlertHandlers(monitor *alerts.Monitor, config *models.AlertConfigResponse) *AlertHandlers {
	return &AlertHandlers{
		monitor: monitor,
		config:  config,
	}
}

// GetAlerts returns the list of recent alerts
func (h *AlertHandlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"alerts":              []models.Alert{},
			"unacknowledgedCount": 0,
		})
		return
	}

	history := h.monitor.GetHistory()
	alerts := history.GetAll()
	unacknowledgedCount := history.GetUnacknowledgedCount()

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"alerts":              alerts,
		"unacknowledgedCount": unacknowledgedCount,
	})
}

// GetAlertConfig returns the current alert configuration
func (h *AlertHandlers) GetAlertConfig(w http.ResponseWriter, r *http.Request) {
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"config": h.config,
	})
}

// AcknowledgeAlert marks an alert as acknowledged
func (h *AlertHandlers) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		http.Error(w, "alerts not enabled", http.StatusNotFound)
		return
	}

	alertID := chi.URLParam(r, "id")
	if alertID == "" {
		http.Error(w, "alert id is required", http.StatusBadRequest)
		return
	}

	history := h.monitor.GetHistory()
	if history.Acknowledge(alertID) {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"message": "Alert acknowledged",
		})
	} else {
		http.Error(w, "alert not found", http.StatusNotFound)
	}
}

// AcknowledgeAllAlerts marks all alerts as acknowledged
func (h *AlertHandlers) AcknowledgeAllAlerts(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		http.Error(w, "alerts not enabled", http.StatusNotFound)
		return
	}

	history := h.monitor.GetHistory()
	alerts := history.GetAll()
	count := 0
	for _, alert := range alerts {
		if !alert.Acknowledged {
			history.Acknowledge(alert.ID)
			count++
		}
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "All alerts acknowledged",
		"count":   count,
	})
}

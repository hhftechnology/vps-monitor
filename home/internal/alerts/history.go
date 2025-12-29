package alerts

import (
	"sync"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// AlertHistory stores recent alerts in memory using a ring buffer
type AlertHistory struct {
	alerts  []models.Alert
	mu      sync.RWMutex
	maxSize int
}

// NewAlertHistory creates a new alert history with the specified max size
func NewAlertHistory(maxSize int) *AlertHistory {
	return &AlertHistory{
		alerts:  make([]models.Alert, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds an alert to the history
func (h *AlertHistory) Add(alert models.Alert) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to the beginning for newest-first ordering
	h.alerts = append([]models.Alert{alert}, h.alerts...)

	// Trim if exceeds max size
	if len(h.alerts) > h.maxSize {
		h.alerts = h.alerts[:h.maxSize]
	}
}

// GetRecent returns the most recent alerts up to the specified limit
func (h *AlertHistory) GetRecent(limit int) []models.Alert {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.alerts) {
		limit = len(h.alerts)
	}

	// Return a copy to avoid data races
	result := make([]models.Alert, limit)
	copy(result, h.alerts[:limit])
	return result
}

// GetAll returns all alerts in history
func (h *AlertHistory) GetAll() []models.Alert {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]models.Alert, len(h.alerts))
	copy(result, h.alerts)
	return result
}

// Acknowledge marks an alert as acknowledged
func (h *AlertHistory) Acknowledge(alertID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.alerts {
		if h.alerts[i].ID == alertID {
			h.alerts[i].Acknowledged = true
			return true
		}
	}
	return false
}

// GetUnacknowledgedCount returns the count of unacknowledged alerts
func (h *AlertHistory) GetUnacknowledgedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, alert := range h.alerts {
		if !alert.Acknowledged {
			count++
		}
	}
	return count
}

// Clear removes all alerts from history
func (h *AlertHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.alerts = make([]models.Alert, 0, h.maxSize)
}

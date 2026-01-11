package models

// AlertType represents the type of alert
type AlertType string

const (
	AlertContainerStopped AlertType = "container_stopped"
	AlertContainerStarted AlertType = "container_started"
	AlertCPUThreshold     AlertType = "cpu_threshold"
	AlertMemoryThreshold  AlertType = "memory_threshold"
)

// Alert represents a system alert
type Alert struct {
	ID            string    `json:"id"`
	Type          AlertType `json:"type"`
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Host          string    `json:"host"`
	Message       string    `json:"message"`
	Value         float64   `json:"value,omitempty"`
	Threshold     float64   `json:"threshold,omitempty"`
	Timestamp     int64     `json:"timestamp"`
	Acknowledged  bool      `json:"acknowledged"`
}

// AlertConfigResponse represents the alert configuration for API responses
type AlertConfigResponse struct {
	Enabled         bool    `json:"enabled"`
	CPUThreshold    float64 `json:"cpu_threshold"`
	MemoryThreshold float64 `json:"memory_threshold"`
	CheckInterval   string  `json:"check_interval"`
	WebhookEnabled  bool    `json:"webhook_enabled"`
}

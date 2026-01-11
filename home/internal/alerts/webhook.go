package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// WebhookPayload is the JSON structure sent to webhooks
type WebhookPayload struct {
	Alert     models.Alert `json:"alert"`
	Timestamp int64        `json:"timestamp"`
	Source    string       `json:"source"`
}

// SendWebhook sends an alert to the configured webhook URL
func SendWebhook(ctx context.Context, url string, alert models.Alert) error {
	if url == "" {
		return nil // No webhook configured
	}

	payload := WebhookPayload{
		Alert:     alert,
		Timestamp: time.Now().Unix(),
		Source:    "vps-monitor",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VPS-Monitor/1.0")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

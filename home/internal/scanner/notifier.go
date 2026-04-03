package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// Notifier sends scan result notifications to Discord and Slack.
type Notifier struct {
	client *http.Client
}

// NewNotifier creates a new notifier.
func NewNotifier() *Notifier {
	return &Notifier{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendDiscord sends a scan result notification to a Discord webhook.
func (n *Notifier) SendDiscord(webhookURL string, result *models.ScanResult, bulkJob *models.BulkScanJob) error {
	var payload map[string]interface{}

	if bulkJob != nil {
		payload = n.buildDiscordBulkPayload(bulkJob)
	} else if result != nil {
		payload = n.buildDiscordScanPayload(result)
	} else {
		return nil
	}

	return n.sendWebhook(webhookURL, payload)
}

// SendSlack sends a scan result notification to a Slack webhook.
func (n *Notifier) SendSlack(webhookURL string, result *models.ScanResult, bulkJob *models.BulkScanJob) error {
	var payload map[string]interface{}

	if bulkJob != nil {
		payload = n.buildSlackBulkPayload(bulkJob)
	} else if result != nil {
		payload = n.buildSlackScanPayload(result)
	} else {
		return nil
	}

	return n.sendWebhook(webhookURL, payload)
}

// SendTestNotification sends a test notification to verify webhook configuration.
func (n *Notifier) SendTestNotification(discordURL, slackURL string) error {
	testResult := &models.ScanResult{
		ImageRef: "test/image:latest",
		Host:     "test-host",
		Scanner:  models.ScannerGrype,
		Summary: models.SeveritySummary{
			Critical: 1,
			High:     3,
			Medium:   5,
			Low:      2,
			Total:    11,
		},
		DurationMs: 5000,
	}

	if discordURL != "" {
		if err := n.SendDiscord(discordURL, testResult, nil); err != nil {
			return fmt.Errorf("discord: %w", err)
		}
	}
	if slackURL != "" {
		if err := n.SendSlack(slackURL, testResult, nil); err != nil {
			return fmt.Errorf("slack: %w", err)
		}
	}
	return nil
}

func (n *Notifier) buildDiscordScanPayload(result *models.ScanResult) map[string]interface{} {
	color := discordColor(result.Summary)
	fields := []map[string]interface{}{
		{"name": "Critical", "value": fmt.Sprintf("%d", result.Summary.Critical), "inline": true},
		{"name": "High", "value": fmt.Sprintf("%d", result.Summary.High), "inline": true},
		{"name": "Medium", "value": fmt.Sprintf("%d", result.Summary.Medium), "inline": true},
		{"name": "Low", "value": fmt.Sprintf("%d", result.Summary.Low), "inline": true},
		{"name": "Total", "value": fmt.Sprintf("%d", result.Summary.Total), "inline": true},
		{"name": "Scanner", "value": string(result.Scanner), "inline": true},
	}
	if result.DurationMs > 0 {
		fields = append(fields, map[string]interface{}{
			"name": "Duration", "value": fmt.Sprintf("%.1fs", float64(result.DurationMs)/1000), "inline": true,
		})
	}

	return map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "Vulnerability Scan Complete",
				"description": fmt.Sprintf("**%s** on host **%s**", result.ImageRef, result.Host),
				"color":       color,
				"fields":      fields,
				"footer":      map[string]string{"text": "VPS Monitor"},
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

func (n *Notifier) buildDiscordBulkPayload(bulkJob *models.BulkScanJob) map[string]interface{} {
	description := fmt.Sprintf("Scanned **%d** images\nCompleted: **%d** | Failed: **%d**",
		bulkJob.TotalImages, bulkJob.Completed, bulkJob.Failed)

	// Aggregate severity counts across all completed scans
	var totalSummary models.SeveritySummary
	for _, job := range bulkJob.Jobs {
		if job.Result != nil {
			totalSummary.Critical += job.Result.Summary.Critical
			totalSummary.High += job.Result.Summary.High
			totalSummary.Medium += job.Result.Summary.Medium
			totalSummary.Low += job.Result.Summary.Low
			totalSummary.Total += job.Result.Summary.Total
		}
	}

	color := discordColor(totalSummary)
	fields := []map[string]interface{}{
		{"name": "Critical", "value": fmt.Sprintf("%d", totalSummary.Critical), "inline": true},
		{"name": "High", "value": fmt.Sprintf("%d", totalSummary.High), "inline": true},
		{"name": "Medium", "value": fmt.Sprintf("%d", totalSummary.Medium), "inline": true},
		{"name": "Low", "value": fmt.Sprintf("%d", totalSummary.Low), "inline": true},
		{"name": "Total", "value": fmt.Sprintf("%d", totalSummary.Total), "inline": true},
	}

	return map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "Bulk Vulnerability Scan Complete",
				"description": description,
				"color":       color,
				"fields":      fields,
				"footer":      map[string]string{"text": "VPS Monitor"},
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

func (n *Notifier) buildSlackScanPayload(result *models.ScanResult) map[string]interface{} {
	summaryText := fmt.Sprintf("Critical: %d | High: %d | Medium: %d | Low: %d | Total: %d",
		result.Summary.Critical, result.Summary.High, result.Summary.Medium,
		result.Summary.Low, result.Summary.Total)

	return map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "Vulnerability Scan Complete",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s* on host *%s*\n\n%s", result.ImageRef, result.Host, summaryText),
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("Scanner: %s | Duration: %.1fs | VPS Monitor", result.Scanner, float64(result.DurationMs)/1000)},
				},
			},
		},
	}
}

func (n *Notifier) buildSlackBulkPayload(bulkJob *models.BulkScanJob) map[string]interface{} {
	var totalSummary models.SeveritySummary
	for _, job := range bulkJob.Jobs {
		if job.Result != nil {
			totalSummary.Critical += job.Result.Summary.Critical
			totalSummary.High += job.Result.Summary.High
			totalSummary.Medium += job.Result.Summary.Medium
			totalSummary.Low += job.Result.Summary.Low
			totalSummary.Total += job.Result.Summary.Total
		}
	}

	summaryText := fmt.Sprintf("Critical: %d | High: %d | Medium: %d | Low: %d | Total: %d",
		totalSummary.Critical, totalSummary.High, totalSummary.Medium,
		totalSummary.Low, totalSummary.Total)

	return map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "Bulk Vulnerability Scan Complete",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("Scanned *%d* images | Completed: *%d* | Failed: *%d*\n\n%s",
						bulkJob.TotalImages, bulkJob.Completed, bulkJob.Failed, summaryText),
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": "VPS Monitor"},
				},
			},
		},
	}
}

func (n *Notifier) sendWebhook(url string, payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VPS-Monitor/1.0")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

// discordColor returns the embed color based on highest severity.
func discordColor(summary models.SeveritySummary) int {
	if summary.Critical > 0 {
		return 0xED4245 // Red
	}
	if summary.High > 0 {
		return 0xED4245 // Red
	}
	if summary.Medium > 0 {
		return 0xFFA500 // Orange
	}
	if summary.Low > 0 {
		return 0xFEE75C // Yellow
	}
	return 0x57F287 // Green - no vulnerabilities
}

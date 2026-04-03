package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
)

// ScanHandlers holds dependencies for scan-related handlers
type ScanHandlers struct {
	scanner *scanner.ScannerService
	manager *config.Manager
}

// NewScanHandlers creates new scan handlers
func NewScanHandlers(scannerService *scanner.ScannerService, manager *config.Manager) *ScanHandlers {
	return &ScanHandlers{
		scanner: scannerService,
		manager: manager,
	}
}

// StartScan handles POST /api/v1/scan
func (h *ScanHandlers) StartScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImageRef string             `json:"imageRef"`
		Host     string             `json:"host"`
		Scanner  models.ScannerType `json:"scanner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ImageRef == "" || req.Host == "" {
		http.Error(w, "imageRef and host are required", http.StatusBadRequest)
		return
	}

	job, err := h.scanner.StartScan(req.ImageRef, req.Host, req.Scanner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"job": job,
	})
}

// StartBulkScan handles POST /api/v1/scan/bulk
func (h *ScanHandlers) StartBulkScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scanner models.ScannerType `json:"scanner"`
		Hosts   []string           `json:"hosts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	bulkJob, err := h.scanner.StartBulkScan(req.Scanner, req.Hosts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"job": bulkJob,
	})
}

// GetScanJobs handles GET /api/v1/scan/jobs
func (h *ScanHandlers) GetScanJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.scanner.GetJobs()
	bulkJobs := h.scanner.GetBulkJobs()

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"jobs":     jobs,
		"bulkJobs": bulkJobs,
	})
}

// GetScanJob handles GET /api/v1/scan/jobs/{id}
func (h *ScanHandlers) GetScanJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	// Check if it's a regular job or bulk job
	job := h.scanner.GetJob(id)
	if job != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"job": job,
		})
		return
	}

	bulkJob := h.scanner.GetBulkJob(id)
	if bulkJob != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"bulkJob": bulkJob,
		})
		return
	}

	http.Error(w, "job not found", http.StatusNotFound)
}

// CancelScanJob handles DELETE /api/v1/scan/jobs/{id}
func (h *ScanHandlers) CancelScanJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	if h.scanner.CancelJob(id) {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"message": "Job cancelled",
		})
	} else {
		http.Error(w, "job not found or already completed", http.StatusNotFound)
	}
}

// GetScanResults handles GET /api/v1/scan/results/{imageRef}
func (h *ScanHandlers) GetScanResults(w http.ResponseWriter, r *http.Request) {
	imageRef, err := url.PathUnescape(chi.URLParam(r, "imageRef"))
	if err != nil {
		http.Error(w, "invalid imageRef", http.StatusBadRequest)
		return
	}
	host := r.URL.Query().Get("host")
	if host == "" {
		http.Error(w, "host query parameter is required", http.StatusBadRequest)
		return
	}

	results := h.scanner.Store().GetResults(host, imageRef)
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

// GetLatestScanResult handles GET /api/v1/scan/results/{imageRef}/latest
func (h *ScanHandlers) GetLatestScanResult(w http.ResponseWriter, r *http.Request) {
	imageRef, err := url.PathUnescape(chi.URLParam(r, "imageRef"))
	if err != nil {
		http.Error(w, "invalid imageRef", http.StatusBadRequest)
		return
	}
	host := r.URL.Query().Get("host")
	if host == "" {
		http.Error(w, "host query parameter is required", http.StatusBadRequest)
		return
	}

	result := h.scanner.Store().GetLatest(host, imageRef)
	if result == nil {
		http.Error(w, "no scan results found", http.StatusNotFound)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

// StartSBOMGeneration handles POST /api/v1/scan/sbom
func (h *ScanHandlers) StartSBOMGeneration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImageRef string           `json:"imageRef"`
		Host     string           `json:"host"`
		Format   models.SBOMFormat `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ImageRef == "" || req.Host == "" {
		http.Error(w, "imageRef and host are required", http.StatusBadRequest)
		return
	}

	if req.Format == "" {
		req.Format = models.SBOMFormatSPDX
	}

	job, err := h.scanner.StartSBOMGeneration(req.ImageRef, req.Host, req.Format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"job": job,
	})
}

// GetSBOMJob handles GET /api/v1/scan/sbom/{id}
func (h *ScanHandlers) GetSBOMJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	job := h.scanner.GetSBOMJob(id)
	if job == nil {
		http.Error(w, "SBOM job not found", http.StatusNotFound)
		return
	}

	// If complete and download requested, serve the file
	if job.Status == models.ScanJobComplete && r.URL.Query().Get("download") == "true" && job.FilePath != "" {
		w.Header().Set("Content-Disposition", "attachment; filename=sbom-"+id+".json")
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, job.FilePath)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"job": job,
	})
}

// GetScannerConfig handles GET /api/v1/settings/scan
func (h *ScanHandlers) GetScannerConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.scanner.Config()
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"config": cfg,
	})
}

// UpdateScannerConfig handles PUT /api/v1/settings/scan
func (h *ScanHandlers) UpdateScannerConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GrypeImage     string `json:"grypeImage"`
		TrivyImage     string `json:"trivyImage"`
		SyftImage      string `json:"syftImage"`
		DefaultScanner string `json:"defaultScanner"`
		GrypeArgs      string `json:"grypeArgs"`
		TrivyArgs      string `json:"trivyArgs"`
		Notifications  struct {
			DiscordWebhookURL string `json:"discordWebhookURL"`
			SlackWebhookURL   string `json:"slackWebhookURL"`
			OnScanComplete    *bool  `json:"onScanComplete"`
			OnBulkComplete    *bool  `json:"onBulkComplete"`
			MinSeverity       string `json:"minSeverity"`
		} `json:"notifications"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Persist to file config
	fileCfg := &config.FileScannerConfig{
		GrypeImage:     req.GrypeImage,
		TrivyImage:     req.TrivyImage,
		SyftImage:      req.SyftImage,
		DefaultScanner: req.DefaultScanner,
		GrypeArgs:      req.GrypeArgs,
		TrivyArgs:      req.TrivyArgs,
		Notifications: &config.FileNotificationConfig{
			DiscordWebhookURL: req.Notifications.DiscordWebhookURL,
			SlackWebhookURL:   req.Notifications.SlackWebhookURL,
			OnScanComplete:    req.Notifications.OnScanComplete,
			OnBulkComplete:    req.Notifications.OnBulkComplete,
			MinSeverity:       req.Notifications.MinSeverity,
		},
	}

	if err := h.manager.UpdateScannerConfig(fileCfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the scanner service config
	mergedCfg := h.manager.Config()
	scannerCfg := configToScannerConfig(&mergedCfg.Scanner)
	h.scanner.UpdateConfig(scannerCfg)

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Scanner configuration updated",
		"config":  scannerCfg,
	})
}

// TestScanNotification handles POST /api/v1/settings/scan/test-notification
func (h *ScanHandlers) TestScanNotification(w http.ResponseWriter, r *http.Request) {
	cfg := h.scanner.Config()

	notifier := scanner.NewNotifier()
	if err := notifier.SendTestNotification(cfg.Notifications.DiscordWebhookURL, cfg.Notifications.SlackWebhookURL); err != nil {
		http.Error(w, "notification test failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Test notification sent successfully",
	})
}

// configToScannerConfig converts config.ScannerConfig to models.ScannerConfig
func configToScannerConfig(cfg *config.ScannerConfig) *models.ScannerConfig {
	return &models.ScannerConfig{
		GrypeImage:     cfg.GrypeImage,
		TrivyImage:     cfg.TrivyImage,
		SyftImage:      cfg.SyftImage,
		DefaultScanner: models.ScannerType(cfg.DefaultScanner),
		GrypeArgs:      cfg.GrypeArgs,
		TrivyArgs:      cfg.TrivyArgs,
		Notifications: models.NotificationConfig{
			DiscordWebhookURL: cfg.DiscordWebhookURL,
			SlackWebhookURL:   cfg.SlackWebhookURL,
			OnScanComplete:    cfg.NotifyOnComplete,
			OnBulkComplete:    cfg.NotifyOnBulk,
			MinSeverity:       models.SeverityLevel(cfg.NotifyMinSeverity),
		},
	}
}

package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
)

// ScanHandlers holds dependencies for scan-related handlers
type ScanHandlers struct {
	scanner          *scanner.ScannerService
	manager          *config.Manager
	autoScanner      *scanner.AutoScanner
	resolveImageIDFn func(host, imageRef string) string
}

// NewScanHandlers creates new scan handlers
func NewScanHandlers(scannerService *scanner.ScannerService, manager *config.Manager) *ScanHandlers {
	return &ScanHandlers{
		scanner: scannerService,
		manager: manager,
	}
}

// SetAutoScanner sets the auto-scanner reference for status reporting.
func (h *ScanHandlers) SetAutoScanner(as *scanner.AutoScanner) {
	h.autoScanner = as
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

	if req.Scanner != "" && req.Scanner != "grype" && req.Scanner != "trivy" {
		http.Error(w, "unsupported scanner, must be 'grype' or 'trivy'", http.StatusBadRequest)
		return
	}

	// Rescan gating: check if image has changed since last scan
	cfg := h.scanner.Config()
	if !cfg.ForceRescan {
		db := h.scanner.Store().DB()
		// Resolve current image ID
		currentImageID := h.resolveImageID(req.Host, req.ImageRef)
		if currentImageID != "" {
			canRescan, err := db.CanRescan(req.Host, req.ImageRef, currentImageID)
			if err != nil {
				log.Printf("Failed to check rescan eligibility: %v", err)
			} else if !canRescan {
				state, _ := db.GetImageScanState(req.Host, req.ImageRef)
				resp := map[string]any{
					"error":   "image_unchanged",
					"message": "Image has not changed since last scan. Pull a new version or enable force rescan in settings.",
				}
				if state != nil {
					resp["last_scan_id"] = state.LastScanID
					resp["last_scan_at"] = state.LastScanAt
				}
				WriteJsonResponse(w, http.StatusConflict, resp)
				return
			}
		}
	}

	job, err := h.scanner.StartScan(req.ImageRef, req.Host, req.Scanner)
	if err != nil {
		log.Printf("Failed to start scan: %v", err)
		http.Error(w, "failed to start scan", http.StatusInternalServerError)
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

	if req.Scanner != "" && req.Scanner != "grype" && req.Scanner != "trivy" {
		http.Error(w, "unsupported scanner, must be 'grype' or 'trivy'", http.StatusBadRequest)
		return
	}

	bulkJob, err := h.scanner.StartBulkScan(req.Scanner, req.Hosts)
	if err != nil {
		log.Printf("Failed to start bulk scan: %v", err)
		http.Error(w, "failed to start bulk scan", http.StatusInternalServerError)
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

// GetScanResults handles GET /api/v1/scan/results
func (h *ScanHandlers) GetScanResults(w http.ResponseWriter, r *http.Request) {
	imageRef := r.URL.Query().Get("image")
	if imageRef == "" {
		http.Error(w, "image query parameter is required", http.StatusBadRequest)
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

// GetLatestScanResult handles GET /api/v1/scan/results/latest
func (h *ScanHandlers) GetLatestScanResult(w http.ResponseWriter, r *http.Request) {
	imageRef := r.URL.Query().Get("image")
	if imageRef == "" {
		http.Error(w, "image query parameter is required", http.StatusBadRequest)
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
		ImageRef string            `json:"imageRef"`
		Host     string            `json:"host"`
		Format   models.SBOMFormat `json:"format"`
		Force    bool              `json:"force"`
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

	cfg := h.scanner.Config()
	if !cfg.ForceRescan && !req.Force {
		db := h.scanner.Store().DB()
		currentImageID := h.resolveImageID(req.Host, req.ImageRef)
		if currentImageID != "" {
			canRegenerate, err := db.CanRegenerateSBOM(req.Host, req.ImageRef, string(req.Format), currentImageID)
			if err != nil {
				log.Printf("Failed to check SBOM regeneration eligibility: %v", err)
			} else if !canRegenerate {
				state, _ := db.GetImageSBOMState(req.Host, req.ImageRef, string(req.Format))
				resp := map[string]any{
					"error":   "image_unchanged",
					"message": "Image has not changed since the last SBOM was generated. Pull a new version or enable force rescan in settings.",
				}
				if state != nil {
					resp["last_sbom_id"] = state.LastSBOMID
					resp["last_sbom_at"] = state.LastSBOMAt
				}
				WriteJsonResponse(w, http.StatusConflict, resp)
				return
			}
		}
	}

	job, err := h.scanner.StartSBOMGeneration(req.ImageRef, req.Host, req.Format)
	if err != nil {
		log.Printf("Failed to start SBOM generation: %v", err)
		http.Error(w, "failed to start SBOM generation", http.StatusInternalServerError)
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
		if _, err := os.Stat(job.FilePath); err != nil {
			http.Error(w, "SBOM file no longer available", http.StatusGone)
			return
		}
		fileID := id
		if job.ResultID != "" {
			fileID = job.ResultID
		}
		w.Header().Set("Content-Disposition", "attachment; filename=sbom-"+fileID+".json")
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
			OnNewCVEs         *bool  `json:"onNewCVEs"`
			MinSeverity       string `json:"minSeverity"`
		} `json:"notifications"`
		AutoScan    *models.AutoScanConfig `json:"autoScan"`
		ForceRescan *bool                  `json:"forceRescan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	validScanners := map[string]bool{"grype": true, "trivy": true}
	if req.DefaultScanner != "" && !validScanners[req.DefaultScanner] {
		http.Error(w, "invalid defaultScanner", http.StatusBadRequest)
		return
	}
	validSeverities := map[string]bool{"Critical": true, "High": true, "Medium": true, "Low": true, "Negligible": true, "Unknown": true}
	if req.Notifications.MinSeverity != "" && !validSeverities[req.Notifications.MinSeverity] {
		http.Error(w, "invalid minSeverity", http.StatusBadRequest)
		return
	}

	// Build the full scanner config to save to DB
	currentCfg := h.scanner.Config()
	newCfg := &models.ScannerConfig{
		GrypeImage:     orDefault(req.GrypeImage, currentCfg.GrypeImage),
		TrivyImage:     orDefault(req.TrivyImage, currentCfg.TrivyImage),
		SyftImage:      orDefault(req.SyftImage, currentCfg.SyftImage),
		DefaultScanner: models.ScannerType(orDefault(req.DefaultScanner, string(currentCfg.DefaultScanner))),
		GrypeArgs:      req.GrypeArgs,
		TrivyArgs:      req.TrivyArgs,
		Notifications: models.NotificationConfig{
			DiscordWebhookURL: req.Notifications.DiscordWebhookURL,
			SlackWebhookURL:   req.Notifications.SlackWebhookURL,
			OnScanComplete:    boolPtrOrDefault(req.Notifications.OnScanComplete, currentCfg.Notifications.OnScanComplete),
			OnBulkComplete:    boolPtrOrDefault(req.Notifications.OnBulkComplete, currentCfg.Notifications.OnBulkComplete),
			OnNewCVEs:         boolPtrOrDefault(req.Notifications.OnNewCVEs, currentCfg.Notifications.OnNewCVEs),
			MinSeverity:       models.SeverityLevel(orDefault(req.Notifications.MinSeverity, string(currentCfg.Notifications.MinSeverity))),
		},
		AutoScan:    currentCfg.AutoScan,
		ForceRescan: currentCfg.ForceRescan,
	}

	if req.AutoScan != nil {
		newCfg.AutoScan = *req.AutoScan
	}
	if req.ForceRescan != nil {
		newCfg.ForceRescan = *req.ForceRescan
	}

	// Save to DB
	db := h.scanner.Store().DB()
	if err := db.SaveScannerSettings(newCfg); err != nil {
		log.Printf("Failed to persist scanner config to DB: %v", err)
		http.Error(w, "failed to update scanner config", http.StatusInternalServerError)
		return
	}

	// Update runtime config
	h.scanner.UpdateConfig(newCfg)

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Scanner configuration updated",
		"config":  newCfg,
	})
}

// TestScanNotification handles POST /api/v1/settings/scan/test-notification
func (h *ScanHandlers) TestScanNotification(w http.ResponseWriter, r *http.Request) {
	cfg := h.scanner.Config()

	notifier := scanner.NewNotifier()
	if err := notifier.SendTestNotification(cfg.Notifications.DiscordWebhookURL, cfg.Notifications.SlackWebhookURL); err != nil {
		log.Printf("Test notification failed: %v", err)
		http.Error(w, "notification test failed", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Test notification sent successfully",
	})
}

// --- History Handlers ---

// GetScanHistory handles GET /api/v1/scan/history
func (h *ScanHandlers) GetScanHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := scanner.HistoryQuery{
		ImageRef:    q.Get("image"),
		Host:        q.Get("host"),
		MinSeverity: q.Get("min_severity"),
		SortBy:      q.Get("sort_by"),
		SortDir:     q.Get("sort_dir"),
	}

	if v := q.Get("page"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid page", http.StatusBadRequest)
			return
		}
		params.Page = val
	}
	if v := q.Get("page_size"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid page_size", http.StatusBadRequest)
			return
		}
		params.PageSize = val
	}
	if v := q.Get("start_date"); v != "" {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid start_date", http.StatusBadRequest)
			return
		}
		params.StartDate = val
	}
	if v := q.Get("end_date"); v != "" {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid end_date", http.StatusBadRequest)
			return
		}
		params.EndDate = val
	}

	db := h.scanner.Store().DB()
	page, err := db.QueryHistory(params)
	if err != nil {
		log.Printf("Failed to query scan history: %v", err)
		http.Error(w, "failed to query scan history", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, page)
}

// GetScanHistoryDetail handles GET /api/v1/scan/history/{id}
func (h *ScanHandlers) GetScanHistoryDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "scan id is required", http.StatusBadRequest)
		return
	}

	db := h.scanner.Store().DB()
	result, err := db.GetResultByID(id)
	if err != nil {
		log.Printf("Failed to get scan result: %v", err)
		http.Error(w, "failed to get scan result", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "scan result not found", http.StatusNotFound)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

// GetScannedImages handles GET /api/v1/scan/history/images
func (h *ScanHandlers) GetScannedImages(w http.ResponseWriter, r *http.Request) {
	db := h.scanner.Store().DB()
	images, err := db.ListScannedImages()
	if err != nil {
		log.Printf("Failed to list scanned images: %v", err)
		http.Error(w, "failed to list scanned images", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"images": images,
	})
}

// GetAutoScanStatus handles GET /api/v1/scan/autoscan/status
func (h *ScanHandlers) GetAutoScanStatus(w http.ResponseWriter, r *http.Request) {
	if h.autoScanner == nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"enabled":         false,
			"lastPollAt":      0,
			"eventsConnected": map[string]bool{},
		})
		return
	}
	WriteJsonResponse(w, http.StatusOK, h.autoScanner.Status())
}

// ExportScanHistory returns a scan history detail in CSV format.
func (h *ScanHandlers) ExportScanHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing scan id", http.StatusBadRequest)
		return
	}

	result, err := h.scanner.Store().DB().GetResultByID(id)
	if err != nil {
		log.Printf("Failed to map export for %s: %v", id, err)
		http.Error(w, "failed to get scan result", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "scan result not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=scan_%s.csv", id))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Severity", "Package", "Version", "VulnerabilityID", "Description", "DataSource"})

	sanitize := func(s string) string {
		if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
			return "'" + s
		}
		return s
	}

	for _, vuln := range result.Vulnerabilities {
		writer.Write([]string{
			string(vuln.Severity),
			sanitize(vuln.Package),
			sanitize(vuln.InstalledVersion),
			sanitize(vuln.ID),
			sanitize(vuln.Description),
			sanitize(vuln.DataSource),
		})
	}
}

// DeleteScanHistory deletes a scan history record from the database.
func (h *ScanHandlers) DeleteScanHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing scan id", http.StatusBadRequest)
		return
	}

	err := h.scanner.Store().DB().DeleteScanResult(id)
	if err != nil {
		http.Error(w, "failed to delete scan result", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSBOMHistory handles GET /api/v1/scan/sbom/history.
func (h *ScanHandlers) GetSBOMHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := scanner.SBOMHistoryQuery{
		ImageRef: q.Get("image"),
		Host:     q.Get("host"),
		Format:   q.Get("format"),
		SortBy:   q.Get("sort_by"),
		SortDir:  q.Get("sort_dir"),
	}

	if v := q.Get("page"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid page", http.StatusBadRequest)
			return
		}
		params.Page = val
	}
	if v := q.Get("page_size"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid page_size", http.StatusBadRequest)
			return
		}
		params.PageSize = val
	}
	if v := q.Get("start_date"); v != "" {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid start_date", http.StatusBadRequest)
			return
		}
		params.StartDate = val
	}
	if v := q.Get("end_date"); v != "" {
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid end_date", http.StatusBadRequest)
			return
		}
		params.EndDate = val
	}

	page, err := h.scanner.Store().DB().QuerySBOMHistory(params)
	if err != nil {
		log.Printf("Failed to query SBOM history: %v", err)
		http.Error(w, "failed to query SBOM history", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, page)
}

// GetSBOMHistoryDetail handles GET /api/v1/scan/sbom/history/{id}.
func (h *ScanHandlers) GetSBOMHistoryDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "sbom id is required", http.StatusBadRequest)
		return
	}

	result, err := h.scanner.Store().DB().GetSBOMResultByID(id)
	if err != nil {
		log.Printf("Failed to get SBOM result: %v", err)
		http.Error(w, "failed to get SBOM result", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "sbom result not found", http.StatusNotFound)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

// GetSBOMedImages handles GET /api/v1/scan/sbom/history/images.
func (h *ScanHandlers) GetSBOMedImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.scanner.Store().DB().ListSBOMedImages()
	if err != nil {
		log.Printf("Failed to list SBOMed images: %v", err)
		http.Error(w, "failed to list SBOMed images", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"images": images,
	})
}

// DownloadSBOMHistory handles GET /api/v1/scan/sbom/history/{id}/download.
func (h *ScanHandlers) DownloadSBOMHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "sbom id is required", http.StatusBadRequest)
		return
	}

	result, err := h.scanner.Store().DB().GetSBOMResultByID(id)
	if err != nil {
		log.Printf("Failed to get SBOM result for download: %v", err)
		http.Error(w, "failed to get SBOM result", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "sbom result not found", http.StatusNotFound)
		return
	}
	if _, err := os.Stat(result.FilePath); err != nil {
		http.Error(w, "SBOM file no longer available", http.StatusGone)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=sbom-"+id+".json")
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, r, result.FilePath)
}

// DeleteSBOMHistory handles DELETE /api/v1/scan/sbom/history/{id}.
func (h *ScanHandlers) DeleteSBOMHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing sbom id", http.StatusBadRequest)
		return
	}

	result, err := h.scanner.Store().DB().GetSBOMResultByID(id)
	if err != nil {
		log.Printf("Failed to load SBOM result for deletion: %v", err)
		http.Error(w, "failed to delete SBOM result", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "sbom result not found", http.StatusNotFound)
		return
	}

	if err := h.scanner.Store().DB().DeleteSBOMResult(id); err != nil {
		log.Printf("Failed to delete SBOM result %s: %v", id, err)
		http.Error(w, "failed to delete SBOM result", http.StatusInternalServerError)
		return
	}

	if err := os.Remove(result.FilePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove SBOM artifact %s: %v", result.FilePath, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func (h *ScanHandlers) resolveImageID(host, imageRef string) string {
	if h.resolveImageIDFn != nil {
		return h.resolveImageIDFn(host, imageRef)
	}

	dockerClient, release := h.scanner.Registry().AcquireDocker()
	if dockerClient == nil {
		release()
		return ""
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiClient, err := dockerClient.GetClient(host)
	if err != nil {
		return ""
	}

	inspect, _, err := apiClient.ImageInspectWithRaw(ctx, imageRef)
	if err != nil {
		return ""
	}
	return inspect.ID
}

func orDefault(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

func boolPtrOrDefault(ptr *bool, def bool) bool {
	if ptr != nil {
		return *ptr
	}
	return def
}

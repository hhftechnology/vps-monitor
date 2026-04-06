package scanner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

const maxConcurrentScansPerHost = 3

// ScannerService orchestrates vulnerability scanning across Docker hosts.
type ScannerService struct {
	registry *services.Registry
	config   atomic.Pointer[models.ScannerConfig]
	store    *ScanResultStore
	notifier *Notifier

	mu       sync.RWMutex
	jobs     map[string]*models.ScanJob
	bulkJobs map[string]*bulkScanState
	sbomJobs map[string]*models.SBOMJob
	cancels  map[string]context.CancelFunc
}

type bulkScanState struct {
	job    *models.BulkScanJob
	cancel context.CancelFunc
}

// NewScannerService creates a new scanner service.
func NewScannerService(registry *services.Registry, cfg *models.ScannerConfig, db *ScanDB) *ScannerService {
	s := &ScannerService{
		registry: registry,
		store:    NewScanResultStore(db),
		notifier: NewNotifier(),
		jobs:     make(map[string]*models.ScanJob),
		bulkJobs: make(map[string]*bulkScanState),
		sbomJobs: make(map[string]*models.SBOMJob),
		cancels:  make(map[string]context.CancelFunc),
	}
	s.config.Store(cfg)
	
	go s.gcWorker()
	
	return s
}

func (s *ScannerService) gcWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		now := time.Now().Unix()
		s.mu.Lock()
		for id, job := range s.jobs {
			if (job.Status == models.ScanJobComplete || job.Status == models.ScanJobFailed || job.Status == models.ScanJobCancelled) && (now-job.CreatedAt > 24*3600) {
				delete(s.jobs, id)
				delete(s.cancels, id)
			}
		}
		for id, state := range s.bulkJobs {
			if (state.job.Status == models.ScanJobComplete || state.job.Status == models.ScanJobFailed || state.job.Status == models.ScanJobCancelled) && (now-state.job.CreatedAt > 24*3600) {
				delete(s.bulkJobs, id)
				delete(s.cancels, id)
			}
		}
		for id, job := range s.sbomJobs {
			if (job.Status == models.ScanJobComplete || job.Status == models.ScanJobFailed || job.Status == models.ScanJobCancelled || job.Status == models.ScanJobExpired) && (now-job.CreatedAt > 24*3600) {
				delete(s.sbomJobs, id)
			}
		}
		s.mu.Unlock()
	}
}

// UpdateConfig updates the scanner configuration.
func (s *ScannerService) UpdateConfig(cfg *models.ScannerConfig) {
	s.config.Store(cfg)
}

// Config returns the current scanner configuration.
func (s *ScannerService) Config() *models.ScannerConfig {
	return s.config.Load()
}

// Store returns the scan result store.
func (s *ScannerService) Store() *ScanResultStore {
	return s.store
}

// Registry returns the service registry.
func (s *ScannerService) Registry() *services.Registry {
	return s.registry
}

// StartScan starts a single image vulnerability scan.
func (s *ScannerService) StartScan(imageRef, host string, scannerType models.ScannerType) (*models.ScanJob, error) {
	cfg := s.Config()
	if scannerType == "" {
		scannerType = cfg.DefaultScanner
	}

	job := &models.ScanJob{
		ID:        uuid.New().String(),
		ImageRef:  imageRef,
		Host:      host,
		Scanner:   scannerType,
		Status:    models.ScanJobPending,
		CreatedAt: time.Now().Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.scanTimeout())

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.cancels[job.ID] = cancel
	s.mu.Unlock()

	go s.runScan(ctx, job, cancel)

	return job, nil
}

// scanTimeout returns the configured per-scan timeout, falling back to 20 minutes.
func (s *ScannerService) scanTimeout() time.Duration {
	if cfg := s.Config(); cfg != nil && cfg.ScanTimeoutMinutes > 0 {
		return time.Duration(cfg.ScanTimeoutMinutes) * time.Minute
	}
	return 20 * time.Minute
}

// bulkTimeout returns the configured bulk-scan timeout, falling back to 120 minutes.
func (s *ScannerService) bulkTimeout() time.Duration {
	if cfg := s.Config(); cfg != nil && cfg.BulkTimeoutMinutes > 0 {
		return time.Duration(cfg.BulkTimeoutMinutes) * time.Minute
	}
	return 120 * time.Minute
}

// scannerLimits returns memory and pids ceilings applied to spawned scanner containers.
func (s *ScannerService) scannerLimits() ScannerLimits {
	memMB := 2048
	pids := int64(512)
	if cfg := s.Config(); cfg != nil {
		if cfg.ScannerMemoryMB > 0 {
			memMB = cfg.ScannerMemoryMB
		}
		if cfg.ScannerPidsLimit > 0 {
			pids = int64(cfg.ScannerPidsLimit)
		}
	}
	return ScannerLimits{
		MemoryBytes: int64(memMB) * 1024 * 1024,
		PidsLimit:   pids,
	}
}

// StartBulkScan starts scanning all images across specified hosts.
func (s *ScannerService) StartBulkScan(scannerType models.ScannerType, hosts []string) (*models.BulkScanJob, error) {
	cfg := s.Config()
	if scannerType == "" {
		scannerType = cfg.DefaultScanner
	}

	// Get all images
	dockerClient, release := s.registry.AcquireDocker()
	if dockerClient == nil {
		release()
		return nil, fmt.Errorf("docker client unavailable")
	}

	ctx := context.Background()
	imagesByHost, _, err := dockerClient.ListImagesAllHosts(ctx)
	release()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	bulkJob := &models.BulkScanJob{
		ID:        uuid.New().String(),
		Status:    models.ScanJobPending,
		CreatedAt: time.Now().Unix(),
	}

	// Create scan jobs for matching hosts
	for hostName, images := range imagesByHost {
		if len(hosts) > 0 && !containsHost(hosts, hostName) {
			continue
		}
		for _, img := range images {
			imageRef := img.ID
			if len(img.RepoTags) > 0 {
				imageRef = img.RepoTags[0]
			}
			job := &models.ScanJob{
				ID:        uuid.New().String(),
				ImageRef:  imageRef,
				Host:      hostName,
				Scanner:   scannerType,
				Status:    models.ScanJobPending,
				CreatedAt: time.Now().Unix(),
			}
			bulkJob.Jobs = append(bulkJob.Jobs, job)

			s.mu.Lock()
			s.jobs[job.ID] = job
			s.mu.Unlock()
		}
	}

	bulkJob.TotalImages = len(bulkJob.Jobs)
	if bulkJob.TotalImages == 0 {
		bulkJob.Status = models.ScanJobComplete
		s.mu.Lock()
		s.bulkJobs[bulkJob.ID] = &bulkScanState{job: bulkJob}
		s.mu.Unlock()
		return bulkJob, nil
	}

	bulkCtx, bulkCancel := context.WithTimeout(context.Background(), s.bulkTimeout())

	s.mu.Lock()
	s.bulkJobs[bulkJob.ID] = &bulkScanState{job: bulkJob, cancel: bulkCancel}
	s.cancels[bulkJob.ID] = bulkCancel
	s.mu.Unlock()

	go s.runBulkScan(bulkCtx, bulkJob, bulkCancel)

	return bulkJob, nil
}

// GetJob returns a scan job by ID.
func (s *ScannerService) GetJob(id string) *models.ScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if job, ok := s.jobs[id]; ok {
		copyJob := *job
		return &copyJob
	}
	return nil
}

// GetBulkJob returns a bulk scan job by ID.
func (s *ScannerService) GetBulkJob(id string) *models.BulkScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.bulkJobs[id]; ok {
		copyJob := *state.job
		return &copyJob
	}
	return nil
}

// GetJobs returns all recent scan jobs.
func (s *ScannerService) GetJobs() []*models.ScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*models.ScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		copyJob := *job
		jobs = append(jobs, &copyJob)
	}
	return jobs
}

// GetBulkJobs returns all bulk scan jobs.
func (s *ScannerService) GetBulkJobs() []*models.BulkScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*models.BulkScanJob, 0, len(s.bulkJobs))
	for _, state := range s.bulkJobs {
		copyJob := *state.job
		jobs = append(jobs, &copyJob)
	}
	return jobs
}

// CancelJob cancels a running scan job.
func (s *ScannerService) CancelJob(id string) bool {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	if ok {
		// Only cancel if the job is still in an active (non-terminal) state
		if job, exists := s.jobs[id]; exists {
			switch job.Status {
			case models.ScanJobPending, models.ScanJobPulling, models.ScanJobScanning:
				// active — allow cancel
			default:
				ok = false
			}
		} else if state, exists := s.bulkJobs[id]; exists {
			switch state.job.Status {
			case models.ScanJobPending, models.ScanJobPulling, models.ScanJobScanning:
				// active — allow cancel
			default:
				ok = false
			}
		}
	}
	s.mu.Unlock()

	if ok {
		cancel()
		s.mu.Lock()
		if job, exists := s.jobs[id]; exists {
			if job.Status == models.ScanJobPending || job.Status == models.ScanJobPulling || job.Status == models.ScanJobScanning {
				job.Status = models.ScanJobCancelled
			}
		}
		if state, exists := s.bulkJobs[id]; exists {
			if state.job.Status == models.ScanJobPending || state.job.Status == models.ScanJobPulling || state.job.Status == models.ScanJobScanning {
				state.job.Status = models.ScanJobCancelled
			}
		}
		delete(s.cancels, id)
		s.mu.Unlock()
		return true
	}
	return false
}

// GetSBOMJob returns an SBOM job by ID.
func (s *ScannerService) GetSBOMJob(id string) *models.SBOMJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if job, ok := s.sbomJobs[id]; ok {
		copyJob := *job
		return &copyJob
	}
	return nil
}

func (s *ScannerService) runScan(ctx context.Context, job *models.ScanJob, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		s.mu.Lock()
		delete(s.cancels, job.ID)
		s.mu.Unlock()
	}()

	cfg := s.Config()

	dockerClient, release := s.registry.AcquireDocker()
	if dockerClient == nil {
		release()
		s.updateJobStatus(job, models.ScanJobFailed, "docker client unavailable")
		return
	}
	defer release()

	apiClient, err := dockerClient.GetClient(job.Host)
	if err != nil {
		s.updateJobStatus(job, models.ScanJobFailed, err.Error())
		return
	}

	s.updateJobProgress(job, models.ScanJobPulling, "Pulling scanner image...")

	startedAt := time.Now()
	var vulns []models.Vulnerability

	onProgress := func(msg string) {
		s.updateJobProgress(job, job.Status, msg)
	}

	s.updateJobProgress(job, models.ScanJobScanning, "Scanning...")

	var bytesWritten int64
	hbCtx, hbCancel := context.WithCancel(ctx)
	go s.heartbeat(hbCtx, job, &bytesWritten, time.Now())
	defer hbCancel()

	limits := s.scannerLimits()

	switch job.Scanner {
	case models.ScannerGrype:
		vulns, err = RunGrypeScan(ctx, apiClient, cfg.GrypeImage, job.ImageRef, cfg.GrypeArgs, job.ID, limits, &bytesWritten, onProgress)
	case models.ScannerTrivy:
		vulns, err = RunTrivyScan(ctx, apiClient, cfg.TrivyImage, job.ImageRef, cfg.TrivyArgs, job.ID, limits, &bytesWritten, onProgress)
	default:
		err = fmt.Errorf("unknown scanner type: %s", job.Scanner)
	}

	completedAt := time.Now()

	if err != nil {
		if ctx.Err() != nil {
			s.updateJobStatus(job, models.ScanJobCancelled, "scan cancelled")
		} else {
			s.updateJobStatus(job, models.ScanJobFailed, err.Error())
		}
		return
	}

	summary := computeSummary(vulns)
	result := models.ScanResult{
		ID:              uuid.New().String(),
		ImageRef:        job.ImageRef,
		Host:            job.Host,
		Scanner:         job.Scanner,
		Vulnerabilities: vulns,
		Summary:         summary,
		StartedAt:       startedAt.Unix(),
		CompletedAt:     completedAt.Unix(),
		DurationMs:      completedAt.Sub(startedAt).Milliseconds(),
	}

	if err := s.store.Add(result); err != nil {
		log.Printf("Failed to store scan result: %v", err)
	}

	s.mu.Lock()
	job.Status = models.ScanJobComplete
	job.Result = &result
	s.mu.Unlock()

	// Send notification if configured
	if cfg.Notifications.OnScanComplete {
		s.sendNotification(&result)
	}

	// Anomaly detection: compare with previous scan for new CVEs
	if cfg.Notifications.OnNewCVEs {
		s.checkAndNotifyAnomalies(&result)
	}
}

func (s *ScannerService) runBulkScan(ctx context.Context, bulkJob *models.BulkScanJob, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		s.mu.Lock()
		delete(s.cancels, bulkJob.ID)
		s.mu.Unlock()
	}()

	s.mu.Lock()
	bulkJob.Status = models.ScanJobScanning
	s.mu.Unlock()

	// Group jobs by host for per-host concurrency limiting
	hostJobs := make(map[string][]*models.ScanJob)
	for _, job := range bulkJob.Jobs {
		hostJobs[job.Host] = append(hostJobs[job.Host], job)
	}

	var wg sync.WaitGroup
	for _, jobs := range hostJobs {
		wg.Add(1)
		go func(jobs []*models.ScanJob) {
			defer wg.Done()
			sem := make(chan struct{}, maxConcurrentScansPerHost)
			var hostWg sync.WaitGroup
			for _, job := range jobs {
				select {
				case <-ctx.Done():
					return
				case sem <- struct{}{}:
				}
				hostWg.Add(1)
				go func(j *models.ScanJob) {
					defer hostWg.Done()
					defer func() { <-sem }()

					jobCtx, jobCancel := context.WithTimeout(ctx, s.scanTimeout())
					s.runScan(jobCtx, j, jobCancel)

					s.mu.Lock()
					if j.Status == models.ScanJobComplete {
						bulkJob.Completed++
					} else if j.Status == models.ScanJobFailed {
						bulkJob.Failed++
					}
					s.mu.Unlock()
				}(job)
			}
			hostWg.Wait()
		}(jobs)
	}

	wg.Wait()

	s.mu.Lock()
	if bulkJob.Status != models.ScanJobCancelled {
		bulkJob.Status = models.ScanJobComplete
	}
	s.mu.Unlock()

	// Send bulk notification
	cfg := s.Config()
	if cfg.Notifications.OnBulkComplete {
		s.sendBulkNotification(bulkJob)
	}
}

func (s *ScannerService) updateJobStatus(job *models.ScanJob, status models.ScanJobStatus, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Status = status
	job.Error = errMsg
}

func (s *ScannerService) updateJobProgress(job *models.ScanJob, status models.ScanJobStatus, progress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Status = status
	job.Progress = progress
}

// heartbeat emits a "Scanning... Xs elapsed, Y output" progress line every 5
// seconds using the live byte counter from the streaming writer. It exits when
// ctx is cancelled (i.e. when the scan returns).
func (s *ScannerService) heartbeat(ctx context.Context, job *models.ScanJob, bytesWritten *int64, started time.Time) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n := atomic.LoadInt64(bytesWritten)
			elapsed := time.Since(started).Round(time.Second)
			msg := fmt.Sprintf("Scanning... %s elapsed, %s output", elapsed, humanBytes(n))
			s.mu.Lock()
			if job.Status == models.ScanJobScanning {
				job.Progress = msg
			}
			s.mu.Unlock()
		}
	}
}

// humanBytes formats a byte count as a human-readable string.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func (s *ScannerService) sendNotification(result *models.ScanResult) {
	cfg := s.Config()
	if !meetsMinSeverity(result.Summary, cfg.Notifications.MinSeverity) {
		return
	}
	if cfg.Notifications.DiscordWebhookURL != "" {
		if err := s.notifier.SendDiscord(cfg.Notifications.DiscordWebhookURL, result, nil); err != nil {
			log.Printf("Failed to send Discord notification: %v", err)
		}
	}
	if cfg.Notifications.SlackWebhookURL != "" {
		if err := s.notifier.SendSlack(cfg.Notifications.SlackWebhookURL, result, nil); err != nil {
			log.Printf("Failed to send Slack notification: %v", err)
		}
	}
}

func (s *ScannerService) checkAndNotifyAnomalies(result *models.ScanResult) {
	db := s.store.DB()
	prevResult, err := db.GetPreviousResult(result.Host, result.ImageRef, result.ID)
	if err != nil {
		log.Printf("Failed to get previous scan result for anomaly detection: %v", err)
		return
	}
	if prevResult == nil {
		return // first scan, nothing to compare
	}

	newVulns := findNewVulnerabilities(prevResult.Vulnerabilities, result.Vulnerabilities)
	if len(newVulns) == 0 {
		return
	}

	cfg := s.Config()
	filtered := filterBySeverity(newVulns, cfg.Notifications.MinSeverity)
	if len(filtered) == 0 {
		return
	}

	diff := computeAnomalyDiff(filtered)
	s.sendAnomalyNotification(result, &diff)
}

func (s *ScannerService) sendAnomalyNotification(result *models.ScanResult, diff *AnomalyDiff) {
	cfg := s.Config()
	if cfg.Notifications.DiscordWebhookURL != "" {
		if err := s.notifier.SendDiscordAnomaly(cfg.Notifications.DiscordWebhookURL, result, diff); err != nil {
			log.Printf("Failed to send Discord anomaly notification: %v", err)
		}
	}
	if cfg.Notifications.SlackWebhookURL != "" {
		if err := s.notifier.SendSlackAnomaly(cfg.Notifications.SlackWebhookURL, result, diff); err != nil {
			log.Printf("Failed to send Slack anomaly notification: %v", err)
		}
	}
}

func (s *ScannerService) sendBulkNotification(bulkJob *models.BulkScanJob) {
	cfg := s.Config()

	filteredJob := &models.BulkScanJob{
		ID:          bulkJob.ID,
		TotalImages: bulkJob.TotalImages,
		Completed:   bulkJob.Completed,
		Failed:      bulkJob.Failed,
		Status:      bulkJob.Status,
		CreatedAt:   bulkJob.CreatedAt,
	}

	for _, job := range bulkJob.Jobs {
		if job.Result != nil {
			if meetsMinSeverity(job.Result.Summary, cfg.Notifications.MinSeverity) {
				filteredJob.Jobs = append(filteredJob.Jobs, job)
			}
		}
	}

	if len(filteredJob.Jobs) == 0 {
		return
	}

	if cfg.Notifications.DiscordWebhookURL != "" {
		if err := s.notifier.SendDiscord(cfg.Notifications.DiscordWebhookURL, nil, filteredJob); err != nil {
			log.Printf("Failed to send Discord bulk notification: %v", err)
		}
	}
	if cfg.Notifications.SlackWebhookURL != "" {
		if err := s.notifier.SendSlack(cfg.Notifications.SlackWebhookURL, nil, filteredJob); err != nil {
			log.Printf("Failed to send Slack bulk notification: %v", err)
		}
	}
}

func computeSummary(vulns []models.Vulnerability) models.SeveritySummary {
	summary := models.SeveritySummary{Total: len(vulns)}
	for _, v := range vulns {
		switch v.Severity {
		case models.SeverityCritical:
			summary.Critical++
		case models.SeverityHigh:
			summary.High++
		case models.SeverityMedium:
			summary.Medium++
		case models.SeverityLow:
			summary.Low++
		case models.SeverityNegligible:
			summary.Negligible++
		default:
			summary.Unknown++
		}
	}
	return summary
}

// meetsMinSeverity checks if the vulnerabilities meet the designated target models.SeverityLevel.
// If minSeverity is empty or unrecognized, it intentionally falls through and returns summary.Total > 0.
// This intentionally matches filterBySeverity in anomaly.go, potentially causing notification flooding if no threshold is set.
func meetsMinSeverity(summary models.SeveritySummary, minSeverity models.SeverityLevel) bool {
	switch minSeverity {
	case models.SeverityCritical:
		return summary.Critical > 0
	case models.SeverityHigh:
		return summary.Critical > 0 || summary.High > 0
	case models.SeverityMedium:
		return summary.Critical > 0 || summary.High > 0 || summary.Medium > 0
	case models.SeverityLow:
		return summary.Critical > 0 || summary.High > 0 || summary.Medium > 0 || summary.Low > 0
	default:
		return summary.Total > 0
	}
}

func containsHost(hosts []string, host string) bool {
	for _, h := range hosts {
		if h == host {
			return true
		}
	}
	return false
}

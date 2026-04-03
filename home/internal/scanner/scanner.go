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
func NewScannerService(registry *services.Registry, cfg *models.ScannerConfig) *ScannerService {
	s := &ScannerService{
		registry: registry,
		store:    NewScanResultStore(),
		notifier: NewNotifier(),
		jobs:     make(map[string]*models.ScanJob),
		bulkJobs: make(map[string]*bulkScanState),
		sbomJobs: make(map[string]*models.SBOMJob),
		cancels:  make(map[string]context.CancelFunc),
	}
	s.config.Store(cfg)
	return s
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.cancels[job.ID] = cancel
	s.mu.Unlock()

	go s.runScan(ctx, job, cancel)

	return job, nil
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
			imageRef := img.RepoTags[0]
			if len(img.RepoTags) == 0 {
				imageRef = img.ID
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
		return bulkJob, nil
	}

	bulkCtx, bulkCancel := context.WithTimeout(context.Background(), 60*time.Minute)

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
	return s.jobs[id]
}

// GetBulkJob returns a bulk scan job by ID.
func (s *ScannerService) GetBulkJob(id string) *models.BulkScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.bulkJobs[id]; ok {
		return state.job
	}
	return nil
}

// GetJobs returns all recent scan jobs.
func (s *ScannerService) GetJobs() []*models.ScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*models.ScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetBulkJobs returns all bulk scan jobs.
func (s *ScannerService) GetBulkJobs() []*models.BulkScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*models.BulkScanJob, 0, len(s.bulkJobs))
	for _, state := range s.bulkJobs {
		jobs = append(jobs, state.job)
	}
	return jobs
}

// CancelJob cancels a running scan job.
func (s *ScannerService) CancelJob(id string) bool {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()

	if ok {
		cancel()
		// Update job status
		s.mu.Lock()
		if job, exists := s.jobs[id]; exists {
			job.Status = models.ScanJobCancelled
		}
		if state, exists := s.bulkJobs[id]; exists {
			state.job.Status = models.ScanJobCancelled
		}
		s.mu.Unlock()
		return true
	}
	return false
}

// GetSBOMJob returns an SBOM job by ID.
func (s *ScannerService) GetSBOMJob(id string) *models.SBOMJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sbomJobs[id]
}

func (s *ScannerService) runScan(ctx context.Context, job *models.ScanJob, cancel context.CancelFunc) {
	defer cancel()

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

	switch job.Scanner {
	case models.ScannerGrype:
		vulns, err = RunGrypeScan(ctx, apiClient, cfg.GrypeImage, job.ImageRef, cfg.GrypeArgs, onProgress)
	case models.ScannerTrivy:
		vulns, err = RunTrivyScan(ctx, apiClient, cfg.TrivyImage, job.ImageRef, cfg.TrivyArgs, onProgress)
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

	s.store.Add(result)

	s.mu.Lock()
	job.Status = models.ScanJobComplete
	job.Result = &result
	s.mu.Unlock()

	// Send notification if configured
	if cfg.Notifications.OnScanComplete {
		s.sendNotification(&result)
	}
}

func (s *ScannerService) runBulkScan(ctx context.Context, bulkJob *models.BulkScanJob, cancel context.CancelFunc) {
	defer cancel()

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

					jobCtx, jobCancel := context.WithTimeout(ctx, 10*time.Minute)
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

func (s *ScannerService) sendBulkNotification(bulkJob *models.BulkScanJob) {
	cfg := s.Config()
	if cfg.Notifications.DiscordWebhookURL != "" {
		if err := s.notifier.SendDiscord(cfg.Notifications.DiscordWebhookURL, nil, bulkJob); err != nil {
			log.Printf("Failed to send Discord bulk notification: %v", err)
		}
	}
	if cfg.Notifications.SlackWebhookURL != "" {
		if err := s.notifier.SendSlack(cfg.Notifications.SlackWebhookURL, nil, bulkJob); err != nil {
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

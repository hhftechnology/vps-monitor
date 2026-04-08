package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/google/uuid"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

const sbomDir = "/data/sbom"

type syftStreamResult struct {
	stderr string
	err    error
}

// StartSBOMGeneration starts SBOM generation for an image.
func (s *ScannerService) StartSBOMGeneration(imageRef, host string, format models.SBOMFormat) (*models.SBOMJob, error) {
	job := &models.SBOMJob{
		ID:        uuid.New().String(),
		ImageRef:  imageRef,
		Host:      host,
		Format:    format,
		Status:    models.ScanJobPending,
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	s.sbomJobs[job.ID] = job
	s.mu.Unlock()

	go s.runSBOMGeneration(job)

	return job, nil
}

func (s *ScannerService) runSBOMGeneration(job *models.SBOMJob) {
	cfg := s.Config()
	ctx, cancel := context.WithTimeout(context.Background(), s.scanTimeout())
	defer cancel()

	dockerClient, release := s.registry.AcquireDocker()
	if dockerClient == nil {
		release()
		s.updateSBOMStatus(job, models.ScanJobFailed, "docker client unavailable")
		return
	}
	defer release()

	apiClient, err := dockerClient.GetClient(job.Host)
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, err.Error())
		return
	}

	s.updateSBOMStatus(job, models.ScanJobPulling, "")

	scannerImage := cfg.SyftImage
	cmd := buildSBOMCmd(job.ImageRef, job.Format)

	if err := PullImageWithProgress(ctx, apiClient, scannerImage, nil); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to pull syft image: %v", err))
		return
	}

	if err := EnsureCacheVolume(ctx, apiClient, ScannerCacheVolumes[ScannerKindSyft].Name); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, err.Error())
		return
	}

	s.updateSBOMStatus(job, models.ScanJobScanning, "")

	resp, err := apiClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, BuildScannerHostConfig(ScannerKindSyft, s.scannerLimits()), nil, nil, "")
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to create syft container: %v", err))
		return
	}
	containerID := resp.ID
	defer apiClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})

	if err := apiClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to start syft container: %v", err))
		return
	}

	if err := os.MkdirAll(sbomDir, 0o750); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to create sbom directory: %v", err))
		return
	}
	filePath := filepath.Join(sbomDir, job.ID+".json")

	streamDone := make(chan syftStreamResult, 1)
	go func() {
		stderr, err := StreamContainerStdoutToFile(ctx, apiClient, containerID, filePath, nil)
		streamDone <- syftStreamResult{stderr: stderr, err: err}
	}()

	statusCh, errCh := apiClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	var streamResult syftStreamResult
	waitStatusCh := statusCh
	waitErrCh := errCh
	waitStreamCh := streamDone

	for waitStatusCh != nil || waitErrCh != nil || waitStreamCh != nil {
		select {
		case err := <-waitErrCh:
			waitErrCh = nil
			waitStatusCh = nil
			if err != nil {
				os.Remove(filePath)
				s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("error waiting for syft: %v", err))
				return
			}
		case status := <-waitStatusCh:
			waitStatusCh = nil
			waitErrCh = nil
			exitCode = status.StatusCode
		case streamResult = <-waitStreamCh:
			waitStreamCh = nil
			if streamCause := syftStreamFailureCause(streamResult); streamCause != nil {
				os.Remove(filePath)
				s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("error streaming syft output: %v", streamCause))
				return
			}
		case <-ctx.Done():
			os.Remove(filePath)
			s.updateSBOMStatus(job, models.ScanJobCancelled, "cancelled")
			return
		}
	}

	if exitCode != 0 {
		tail := streamResult.stderr
		if tail == "" {
			tail = readFilePrefix(filePath, 2*1024)
		}
		os.Remove(filePath)
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("syft exited with code %d: %s", exitCode, tail))
		return
	}

	s.mu.Lock()
	job.Status = models.ScanJobComplete
	job.FilePath = filePath
	s.mu.Unlock()

	// Schedule cleanup after 1 hour
	time.AfterFunc(1*time.Hour, func() {
		os.Remove(filePath)
		s.mu.Lock()
		job.FilePath = ""
		job.Status = models.ScanJobExpired
		s.mu.Unlock()
	})
}

func (s *ScannerService) updateSBOMStatus(job *models.SBOMJob, status models.ScanJobStatus, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Status = status
	job.Error = errMsg
}

func buildSBOMCmd(imageRef string, format models.SBOMFormat) []string {
	outputFormat := "spdx-json"
	if format == models.SBOMFormatCycloneDX {
		outputFormat = "cyclonedx-json"
	}
	return []string{imageRef, "-o", outputFormat}
}

func syftStreamFailureCause(result syftStreamResult) any {
	if result.err != nil {
		return result.err
	}
	if stderr := strings.TrimSpace(result.stderr); stderr != "" {
		return stderr
	}
	return nil
}

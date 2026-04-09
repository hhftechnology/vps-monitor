package scanner

import (
	"context"
	"encoding/json"
	"errors"
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

	startedAt := time.Now()
	resultID := uuid.New().String()
	filePath := filepath.Join(sbomDir, resultID+".json")

	inspect, _, err := apiClient.ImageInspectWithRaw(ctx, job.ImageRef)
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to inspect image: %v", err))
		return
	}
	imageID := inspect.ID

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
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				s.updateSBOMStatus(job, models.ScanJobFailed, "scan timed out")
			} else {
				s.updateSBOMStatus(job, models.ScanJobCancelled, "scan cancelled")
			}
			return
		}
	}

	if exitCode != 0 {
		tail := strings.TrimSpace(streamResult.stderr)
		if tail == "" {
			tail = readFilePrefix(filePath, 2*1024)
		}
		os.Remove(filePath)
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("syft exited with code %d: %s", exitCode, tail))
		return
	}

	components, err := parseSBOMComponents(filePath, job.Format)
	if err != nil {
		os.Remove(filePath)
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to parse SBOM: %v", err))
		return
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		os.Remove(filePath)
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to stat SBOM output: %v", err))
		return
	}

	completedAt := time.Now()
	result := models.SBOMResult{
		ID:             resultID,
		ImageRef:       job.ImageRef,
		Host:           job.Host,
		Format:         job.Format,
		ComponentCount: len(components),
		FileSize:       fileInfo.Size(),
		FilePath:       filePath,
		StartedAt:      startedAt.Unix(),
		CompletedAt:    completedAt.Unix(),
		DurationMs:     completedAt.Sub(startedAt).Milliseconds(),
		Components:     components,
	}

	if err := s.store.AddSBOM(result, imageID); err != nil {
		os.Remove(filePath)
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to persist SBOM: %v", err))
		return
	}

	s.mu.Lock()
	job.Status = models.ScanJobComplete
	job.ResultID = resultID
	job.FilePath = filePath
	s.mu.Unlock()
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

type cyclonedxSBOM struct {
	Components []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
		PURL    string `json:"purl"`
	} `json:"components"`
}

type spdxSBOM struct {
	Packages []struct {
		Name         string `json:"name"`
		VersionInfo  string `json:"versionInfo"`
		ExternalRefs []struct {
			ReferenceType    string `json:"referenceType"`
			ReferenceLocator string `json:"referenceLocator"`
		} `json:"externalRefs"`
	} `json:"packages"`
}

func parseSBOMComponents(filePath string, format models.SBOMFormat) ([]models.SBOMComponent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	switch format {
	case models.SBOMFormatCycloneDX:
		var document cyclonedxSBOM
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, err
		}

		components := make([]models.SBOMComponent, 0, len(document.Components))
		for _, component := range document.Components {
			components = append(components, models.SBOMComponent{
				Name:    component.Name,
				Version: component.Version,
				Type:    component.Type,
				PURL:    component.PURL,
			})
		}
		return components, nil
	case models.SBOMFormatSPDX:
		var document spdxSBOM
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, err
		}

		components := make([]models.SBOMComponent, 0, len(document.Packages))
		for _, pkg := range document.Packages {
			purl := ""
			for _, ref := range pkg.ExternalRefs {
				if ref.ReferenceType == "purl" {
					purl = ref.ReferenceLocator
					break
				}
			}
			components = append(components, models.SBOMComponent{
				Name:    pkg.Name,
				Version: pkg.VersionInfo,
				Type:    "package",
				PURL:    purl,
			})
		}
		return components, nil
	default:
		return nil, fmt.Errorf("unsupported SBOM format: %s", format)
	}
}

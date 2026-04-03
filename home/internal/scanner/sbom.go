package scanner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

const sbomDir = "/data/sbom"

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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

	// Use Syft for SBOM generation
	scannerImage := cfg.SyftImage
	cmd := buildSBOMCmd(job.ImageRef, job.Format)

	// Pull scanner image
	pullReader, err := apiClient.ImagePull(ctx, scannerImage, image.PullOptions{})
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to pull syft image: %v", err))
		return
	}
	io.Copy(io.Discard, pullReader)
	pullReader.Close()

	s.updateSBOMStatus(job, models.ScanJobScanning, "")

	// Create and run container
	resp, err := apiClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, &container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}, nil, nil, "")
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to create syft container: %v", err))
		return
	}
	containerID := resp.ID
	defer apiClient.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})

	if err := apiClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to start syft container: %v", err))
		return
	}

	// Wait for completion
	statusCh, errCh := apiClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("error waiting for syft: %v", err))
			return
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			logs, _ := getContainerLogs(ctx, apiClient, containerID)
			s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("syft exited with code %d: %s", status.StatusCode, logs))
			return
		}
	case <-ctx.Done():
		s.updateSBOMStatus(job, models.ScanJobCancelled, "cancelled")
		return
	}

	// Read output and save to file
	logReader, err := apiClient.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to read syft output: %v", err))
		return
	}
	defer logReader.Close()

	output, err := demuxDockerLogs(logReader)
	if err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to read syft output: %v", err))
		return
	}

	// Write SBOM to file
	if err := os.MkdirAll(sbomDir, 0750); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to create sbom directory: %v", err))
		return
	}

	filePath := filepath.Join(sbomDir, job.ID+".json")
	if err := os.WriteFile(filePath, output, 0600); err != nil {
		s.updateSBOMStatus(job, models.ScanJobFailed, fmt.Sprintf("failed to write sbom file: %v", err))
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

// RunSBOMWithTrivy generates an SBOM using Trivy instead of Syft.
func RunSBOMWithTrivy(ctx context.Context, dockerClient *client.Client, trivyImage, imageRef string, format models.SBOMFormat) ([]byte, error) {
	outputFormat := "spdx-json"
	if format == models.SBOMFormatCycloneDX {
		outputFormat = "cyclonedx"
	}

	cmd := []string{"image", "--format", outputFormat, imageRef}

	pullReader, err := dockerClient.ImagePull(ctx, trivyImage, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull trivy image: %w", err)
	}
	io.Copy(io.Discard, pullReader)
	pullReader.Close()

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: trivyImage,
		Cmd:   cmd,
	}, &container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create trivy sbom container: %w", err)
	}
	containerID := resp.ID
	defer dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})

	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start trivy sbom container: %w", err)
	}

	statusCh, errCh := dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for trivy sbom: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			logs, _ := getContainerLogs(ctx, dockerClient, containerID)
			return nil, fmt.Errorf("trivy sbom exited with code %d: %s", status.StatusCode, logs)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	logReader, err := dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, fmt.Errorf("failed to read trivy sbom output: %w", err)
	}
	defer logReader.Close()

	return demuxDockerLogs(logReader)
}

// getContainerLogs is defined in grype.go, avoid redeclaration by using the existing one.
// demuxDockerLogs is defined in grype.go, shared across the package.
// normalizeSeverity is defined in grype.go, shared across the package.

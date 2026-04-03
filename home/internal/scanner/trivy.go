package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// trivyOutput represents the JSON output structure from Trivy
type trivyOutput struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target          string              `json:"Target"`
	Vulnerabilities []trivyVulnerability `json:"Vulnerabilities"`
}

type trivyVulnerability struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Description      string `json:"Description"`
	PrimaryURL       string `json:"PrimaryURL"`
}

// RunTrivyScan runs a Trivy vulnerability scan against an image using Docker.
func RunTrivyScan(ctx context.Context, dockerClient *client.Client, scannerImage, imageRef, args string, onProgress func(string)) ([]models.Vulnerability, error) {
	// Pull the scanner image
	if onProgress != nil {
		onProgress("Pulling scanner image " + scannerImage + "...")
	}
	pullReader, err := dockerClient.ImagePull(ctx, scannerImage, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull trivy image: %w", err)
	}
	io.Copy(io.Discard, pullReader)
	pullReader.Close()

	// Build the command
	cmd := buildTrivyCmd(imageRef, args)

	if onProgress != nil {
		onProgress("Scanning " + imageRef + " with Trivy...")
	}

	// Create and start scanner container
	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, &container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create trivy container: %w", err)
	}
	containerID := resp.ID
	defer dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})

	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start trivy container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for trivy container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			logs, _ := getContainerLogs(ctx, dockerClient, containerID)
			return nil, fmt.Errorf("trivy exited with code %d: %s", status.StatusCode, logs)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Read stdout for JSON output
	logReader, err := dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, fmt.Errorf("failed to read trivy output: %w", err)
	}
	defer logReader.Close()

	output, err := demuxDockerLogs(logReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read trivy output: %w", err)
	}

	return parseTrivyOutput(output)
}

// buildTrivyCmd constructs the command for Trivy.
func buildTrivyCmd(imageRef, args string) []string {
	if args != "" {
		resolved := strings.ReplaceAll(args, "{image}", imageRef)
		return strings.Fields(resolved)
	}
	return []string{"image", "--format", "json", imageRef}
}

// parseTrivyOutput parses Trivy JSON output into vulnerabilities.
func parseTrivyOutput(data []byte) ([]models.Vulnerability, error) {
	var output trivyOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
	}

	var vulns []models.Vulnerability
	for _, result := range output.Results {
		for _, v := range result.Vulnerabilities {
			vulns = append(vulns, models.Vulnerability{
				ID:               v.VulnerabilityID,
				Severity:         normalizeSeverity(v.Severity),
				Package:          v.PkgName,
				InstalledVersion: v.InstalledVersion,
				FixedVersion:     v.FixedVersion,
				Description:      v.Description,
				DataSource:       v.PrimaryURL,
			})
		}
	}

	if vulns == nil {
		vulns = []models.Vulnerability{}
	}

	return vulns, nil
}

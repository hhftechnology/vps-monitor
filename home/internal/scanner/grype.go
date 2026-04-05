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
	"github.com/google/shlex"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// grypeOutput represents the JSON output structure from Grype
type grypeOutput struct {
	Matches []grypeMatch `json:"matches"`
}

type grypeMatch struct {
	Vulnerability grypeVulnerability `json:"vulnerability"`
	Artifact      grypeArtifact      `json:"artifact"`
}

type grypeVulnerability struct {
	ID          string          `json:"id"`
	Severity    string          `json:"severity"`
	Description string          `json:"description"`
	DataSource  string          `json:"dataSource"`
	Fix         grypeFixInfo    `json:"fix"`
}

type grypeFixInfo struct {
	Versions []string `json:"versions"`
	State    string   `json:"state"`
}

type grypeArtifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RunGrypeScan runs a Grype vulnerability scan against an image using Docker.
func RunGrypeScan(ctx context.Context, dockerClient *client.Client, scannerImage, imageRef, args string, onProgress func(string)) ([]models.Vulnerability, error) {
	// Pull the scanner image
	if onProgress != nil {
		onProgress("Pulling scanner image " + scannerImage + "...")
	}
	pullReader, err := dockerClient.ImagePull(ctx, scannerImage, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull grype image: %w", err)
	}
	io.Copy(io.Discard, pullReader)
	pullReader.Close()

	// Build the command
	cmd := buildGrypeCmd(imageRef, args)

	if onProgress != nil {
		onProgress("Scanning " + imageRef + " with Grype...")
	}

	// Create and start scanner container
	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, &container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create grype container: %w", err)
	}
	containerID := resp.ID
	defer dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})

	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start grype container: %w", err)
	}

	// Wait for completion
	statusCh, errCh := dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for grype container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			// Grype exits with code 1 when vulnerabilities are found - that's expected
			if status.StatusCode != 1 {
				logs, _ := getContainerLogs(ctx, dockerClient, containerID)
				return nil, fmt.Errorf("grype exited with code %d: %s", status.StatusCode, logs)
			}
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Read stdout for JSON output
	logReader, err := dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, fmt.Errorf("failed to read grype output: %w", err)
	}
	defer logReader.Close()

	output, err := demuxDockerLogs(logReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read grype output: %w", err)
	}

	return parseGrypeOutput(output)
}

// buildGrypeCmd constructs the command for Grype.
func buildGrypeCmd(imageRef, args string) []string {
	if args != "" {
		resolved := strings.ReplaceAll(args, "{image}", imageRef)
		parts, err := shlex.Split(resolved)
		if err != nil {
			return strings.Fields(resolved)
		}
		return parts
	}
	return []string{imageRef, "-o", "json"}
}

// parseGrypeOutput parses Grype JSON output into vulnerabilities.
func parseGrypeOutput(data []byte) ([]models.Vulnerability, error) {
	var output grypeOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse grype output: %w", err)
	}

	vulns := make([]models.Vulnerability, 0, len(output.Matches))
	for _, match := range output.Matches {
		fixedVersion := ""
		if len(match.Vulnerability.Fix.Versions) > 0 {
			fixedVersion = match.Vulnerability.Fix.Versions[0]
		}

		vulns = append(vulns, models.Vulnerability{
			ID:               match.Vulnerability.ID,
			Severity:         normalizeSeverity(match.Vulnerability.Severity),
			Package:          match.Artifact.Name,
			InstalledVersion: match.Artifact.Version,
			FixedVersion:     fixedVersion,
			Description:      match.Vulnerability.Description,
			DataSource:       match.Vulnerability.DataSource,
		})
	}

	return vulns, nil
}

// getContainerLogs reads stderr from a container for error reporting.
func getContainerLogs(ctx context.Context, dockerClient *client.Client, containerID string) (string, error) {
	reader, err := dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStderr: true})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := demuxDockerLogs(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// normalizeSeverity normalizes severity strings from scanners.
func normalizeSeverity(severity string) models.SeverityLevel {
	switch strings.ToLower(severity) {
	case "critical":
		return models.SeverityCritical
	case "high":
		return models.SeverityHigh
	case "medium":
		return models.SeverityMedium
	case "low":
		return models.SeverityLow
	case "negligible":
		return models.SeverityNegligible
	default:
		return models.SeverityUnknown
	}
}

// demuxDockerLogs reads Docker multiplexed log output and returns the raw content.
func demuxDockerLogs(reader io.Reader) ([]byte, error) {
	// Docker container logs use a multiplexed format with an 8-byte header per frame.
	// Header: [1 byte stream type][3 bytes padding][4 bytes uint32 big-endian size]
	var result []byte
	header := make([]byte, 8)

	for {
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return result, err
		}

		size := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])
		if size == 0 {
			continue
		}

		frame := make([]byte, size)
		_, err = io.ReadFull(reader, frame)
		if err != nil {
			return result, err
		}

		// Only capture stdout (stream type 1)
		if header[0] == 1 {
			result = append(result, frame...)
		}
	}

	return result, nil
}

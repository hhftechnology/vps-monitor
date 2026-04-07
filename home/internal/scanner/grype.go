package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
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
	ID          string       `json:"id"`
	Severity    string       `json:"severity"`
	Description string       `json:"description"`
	DataSource  string       `json:"dataSource"`
	Fix         grypeFixInfo `json:"fix"`
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
// Stdout is streamed directly to disk to keep the backend RSS flat regardless
// of how many vulnerabilities are reported.
func RunGrypeScan(
	ctx context.Context,
	dockerClient *client.Client,
	scannerImage, imageRef, args, jobID string,
	limits ScannerLimits,
	bytesWritten *int64,
	onProgress func(string),
) ([]models.Vulnerability, error) {
	if onProgress != nil {
		onProgress("Pulling scanner image " + scannerImage + "...")
	}
	if err := PullImageWithProgress(ctx, dockerClient, scannerImage, onProgress); err != nil {
		return nil, fmt.Errorf("failed to pull grype image: %w", err)
	}

	if err := EnsureCacheVolume(ctx, dockerClient, ScannerCacheVolumes[ScannerKindGrype].Name); err != nil {
		return nil, err
	}

	cmd := buildGrypeCmd(imageRef, args)

	if onProgress != nil {
		onProgress("Scanning " + imageRef + " with Grype...")
	}

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, BuildScannerHostConfig(ScannerKindGrype, limits), nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create grype container: %w", err)
	}
	containerID := resp.ID
	defer dockerClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})

	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start grype container: %w", err)
	}

	outPath := TempScanFile(jobID)
	defer os.Remove(outPath)

	// Stream logs concurrently with the wait so the file is fully populated by
	// the time the container exits — Docker holds them in memory otherwise.
	streamDone := make(chan struct {
		stderr string
		err    error
	}, 1)
	go func() {
		stderr, err := StreamContainerStdoutToFile(ctx, dockerClient, containerID, outPath, bytesWritten)
		streamDone <- struct {
			stderr string
			err    error
		}{stderr, err}
	}()

	statusCh, errCh := dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for grype container: %w", err)
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	streamResult := <-streamDone
	if streamResult.err != nil && exitCode == 0 {
		return nil, fmt.Errorf("failed to read grype output: %w", streamResult.err)
	}

	// Grype exits with code 1 when vulnerabilities are found — that's expected.
	if exitCode != 0 && exitCode != 1 {
		tail := streamResult.stderr
		if tail == "" {
			tail = readFilePrefix(outPath, 2*1024)
		}
		return nil, fmt.Errorf("grype exited with code %d: %s", exitCode, tail)
	}

	f, err := os.Open(outPath)
	if err != nil {
		return nil, fmt.Errorf("open grype output: %w", err)
	}
	defer f.Close()

	vulns, err := parseGrypeOutputStream(f)
	if err != nil {
		return nil, enrichParseErrorForEmptyOutput("grype", outPath, streamResult.stderr, err)
	}
	return vulns, nil
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

// parseGrypeOutputStream stream-decodes Grype JSON into vulnerabilities.
func parseGrypeOutputStream(r io.Reader) ([]models.Vulnerability, error) {
	var output grypeOutput
	if err := json.NewDecoder(r).Decode(&output); err != nil {
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

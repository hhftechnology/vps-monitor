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

// trivyOutput represents the JSON output structure from Trivy
type trivyOutput struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target          string               `json:"Target"`
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
// Stdout is streamed to disk so the backend RSS stays flat for very large reports.
func RunTrivyScan(
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
		return nil, fmt.Errorf("failed to pull trivy image: %w", err)
	}

	if err := EnsureCacheVolume(ctx, dockerClient, ScannerCacheVolumes[ScannerKindTrivy].Name); err != nil {
		return nil, err
	}

	cmd, err := buildTrivyCmd(imageRef, args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trivy args: %w", err)
	}

	if onProgress != nil {
		onProgress("Scanning " + imageRef + " with Trivy...")
	}

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: scannerImage,
		Cmd:   cmd,
	}, BuildScannerHostConfig(ScannerKindTrivy, limits), nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create trivy container: %w", err)
	}
	containerID := resp.ID
	defer dockerClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})

	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start trivy container: %w", err)
	}

	outPath := TempScanFile(jobID)
	defer os.Remove(outPath)

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
			return nil, fmt.Errorf("error waiting for trivy container: %w", err)
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	streamResult := <-streamDone
	if streamResult.err != nil && exitCode == 0 {
		return nil, fmt.Errorf("failed to read trivy output: %w", streamResult.err)
	}

	if exitCode != 0 {
		tail := streamResult.stderr
		if tail == "" {
			tail = readFilePrefix(outPath, 2*1024)
		}
		return nil, fmt.Errorf("trivy exited with code %d: %s", exitCode, tail)
	}

	f, err := os.Open(outPath)
	if err != nil {
		return nil, fmt.Errorf("open trivy output: %w", err)
	}
	defer f.Close()
	return parseTrivyOutputStream(f)
}

// buildTrivyCmd constructs the command for Trivy.
func buildTrivyCmd(imageRef, args string) ([]string, error) {
	if args != "" {
		resolved := strings.ReplaceAll(args, "{image}", imageRef)
		return shlex.Split(resolved)
	}
	return []string{"image", "--format", "json", imageRef}, nil
}

// parseTrivyOutputStream stream-decodes Trivy JSON into vulnerabilities.
func parseTrivyOutputStream(r io.Reader) ([]models.Vulnerability, error) {
	var output trivyOutput
	if err := json.NewDecoder(r).Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
	}

	vulns := make([]models.Vulnerability, 0)
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

	return vulns, nil
}

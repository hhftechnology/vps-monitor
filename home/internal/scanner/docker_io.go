package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ScannerKind identifies a scanner image kind for cache-volume selection.
type ScannerKind string

const (
	ScannerKindGrype ScannerKind = "grype"
	ScannerKindTrivy ScannerKind = "trivy"
	ScannerKindSyft  ScannerKind = "syft"
)

// CacheVolume describes a Docker named volume mounted into a scanner container.
type CacheVolume struct {
	Name      string
	MountPath string
}

// ScannerCacheVolumes maps each scanner kind to its persistent DB cache volume.
// Mount paths are the in-container directories where the scanners persist their
// vulnerability databases. They assume the official upstream images, which run
// as root by default.
var ScannerCacheVolumes = map[ScannerKind]CacheVolume{
	ScannerKindGrype: {Name: "vps-monitor-grype-db", MountPath: "/root/.cache/grype"},
	ScannerKindTrivy: {Name: "vps-monitor-trivy-db", MountPath: "/root/.cache/trivy"},
	ScannerKindSyft:  {Name: "vps-monitor-syft-db", MountPath: "/root/.cache/syft"},
}

// ScannerLimits captures resource ceilings applied to spawned scanner containers.
type ScannerLimits struct {
	MemoryBytes int64
	PidsLimit   int64
}

// EnsureCacheVolume creates the named volume if it does not already exist.
// VolumeCreate is idempotent server-side, so calling it on every scan is safe.
func EnsureCacheVolume(ctx context.Context, dockerClient *client.Client, name string) error {
	_, err := dockerClient.VolumeCreate(ctx, volume.CreateOptions{Name: name})
	if err != nil {
		return fmt.Errorf("ensure cache volume %s: %w", name, err)
	}
	return nil
}

// BuildScannerHostConfig builds a HostConfig that mounts the docker socket plus
// the appropriate scanner cache volume and applies the supplied resource limits.
func BuildScannerHostConfig(kind ScannerKind, limits ScannerLimits) *container.HostConfig {
	hc := &container.HostConfig{
		Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
	}
	if v, ok := ScannerCacheVolumes[kind]; ok {
		hc.Mounts = nil
		hc.Binds = append(hc.Binds, v.Name+":"+v.MountPath)
	}
	if limits.MemoryBytes > 0 {
		hc.Resources.Memory = limits.MemoryBytes
	}
	if limits.PidsLimit > 0 {
		pids := limits.PidsLimit
		hc.Resources.PidsLimit = &pids
	}
	return hc
}

// pullProgressEvent is a subset of the JSON events returned by ImagePull.
type pullProgressEvent struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Error string `json:"error"`
}

// PullImageWithProgress pulls a Docker image and forwards human-readable
// progress strings to onProgress (throttled to ~1/sec). Returns the first
// error reported in the event stream, surfacing failures that the previous
// io.Copy(io.Discard, …) approach silently swallowed.
func PullImageWithProgress(ctx context.Context, dockerClient *client.Client, ref string, onProgress func(string)) error {
	reader, err := dockerClient.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull %s: %w", ref, err)
	}
	defer reader.Close()

	dec := json.NewDecoder(reader)
	var lastEmit time.Time
	for {
		var ev pullProgressEvent
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("pull %s: decode: %w", ref, err)
		}
		if ev.Error != "" {
			return fmt.Errorf("pull %s: %s", ref, ev.Error)
		}
		if onProgress == nil {
			continue
		}
		if time.Since(lastEmit) < time.Second {
			continue
		}
		lastEmit = time.Now()

		msg := "Pulling " + ref
		if ev.Status != "" {
			msg += ": " + ev.Status
		}
		if ev.ProgressDetail.Total > 0 {
			pct := (ev.ProgressDetail.Current * 100) / ev.ProgressDetail.Total
			msg += fmt.Sprintf(" %d%%", pct)
		}
		onProgress(msg)
	}
}

// countingWriter wraps an io.Writer and atomically increments a counter on
// every successful write. Used to drive the progress heartbeat.
type countingWriter struct {
	w io.Writer
	n *int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 && c.n != nil {
		atomic.AddInt64(c.n, int64(n))
	}
	return n, err
}

// ringBuffer is a fixed-capacity sink that retains the last N bytes written.
// Used for stderr tail capture without unbounded memory growth.
type ringBuffer struct {
	mu  sync.Mutex
	buf []byte
	cap int
}

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ringBuffer{cap: capacity, buf: make([]byte, 0, capacity)}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.cap {
		// Allocate a fresh r.cap-sized slice and copy the trailing bytes so
		// the previous (potentially much larger) backing array can be GC'd.
		newBuf := make([]byte, r.cap)
		copy(newBuf, r.buf[len(r.buf)-r.cap:])
		r.buf = newBuf
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.buf)
}

// streamLogs demultiplexes a Docker container log stream, writing stdout to
// outPath and capturing the tail of stderr in an in-memory ring buffer.
// Total stdout bytes are atomically tracked through bytes (may be nil).
func streamLogs(rc io.Reader, outPath string, bytes *int64) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return "", fmt.Errorf("create out dir: %w", err)
	}
	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("create out file: %w", err)
	}
	defer f.Close()

	stderrTail := newRingBuffer(8 * 1024)
	stdoutSink := &countingWriter{w: f, n: bytes}

	if _, err := stdcopy.StdCopy(stdoutSink, stderrTail, rc); err != nil {
		return stderrTail.String(), fmt.Errorf("stdcopy: %w", err)
	}
	return stderrTail.String(), nil
}

func scannerContainerLogsOptions() container.LogsOptions {
	return container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
}

// StreamContainerStdoutToFile streams a container's logs into outPath, returning
// the captured stderr tail. Stdout is written byte-for-byte to disk so downstream
// parsers can stream-decode without buffering the whole output in memory.
func StreamContainerStdoutToFile(ctx context.Context, dockerClient *client.Client, containerID, outPath string, bytes *int64) (string, error) {
	logs, err := dockerClient.ContainerLogs(ctx, containerID, scannerContainerLogsOptions())
	if err != nil {
		return "", fmt.Errorf("container logs: %w", err)
	}
	defer logs.Close()
	return streamLogs(logs, outPath, bytes)
}

func enrichParseErrorForEmptyOutput(scannerName, outPath, stderrTail string, parseErr error) error {
	if !errors.Is(parseErr, io.EOF) {
		return parseErr
	}

	info, err := os.Stat(outPath)
	if err != nil || info.Size() > 0 {
		return parseErr
	}

	tail := strings.Join(strings.Fields(stderrTail), " ")
	if len(tail) > 512 {
		tail = tail[:512]
	}
	if tail == "" {
		return fmt.Errorf("%w (empty %s output)", parseErr, scannerName)
	}
	return fmt.Errorf("%w (empty %s output; stderr tail: %s)", parseErr, scannerName, tail)
}

// TempScanFile returns the on-disk path used for streaming a scanner's stdout.
func TempScanFile(jobID string) string {
	return filepath.Join(os.TempDir(), "vps-monitor-scan-"+jobID+".json")
}

// readFilePrefix reads up to n bytes from the start of path, used for error reporting.
func readFilePrefix(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, n)
	read, _ := io.ReadFull(f, buf)
	return strings.TrimSpace(string(buf[:read]))
}

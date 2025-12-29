package docker

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// parseDockerLogs parses the Docker log stream into structured entries
func parseDockerLogs(reader io.Reader) ([]models.LogEntry, error) {
	var entries []models.LogEntry

	stdout := &logWriter{stream: "stdout", entries: &entries}
	stderr := &logWriter{stream: "stderr", entries: &entries}

	_, err := stdcopy.StdCopy(stdout, stderr, reader)
	if err != nil && err != io.EOF {
		return nil, err
	}

	stdout.Flush()
	stderr.Flush()

	return entries, nil
}

// logWriter implements io.Writer and parses log lines
type logWriter struct {
	stream  string
	entries *[]models.LogEntry
	buffer  []byte
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)

	for {
		idx := -1
		for i, b := range w.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}

		if idx == -1 {
			break
		}

		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]

		if line != "" {
			line = strings.TrimSuffix(line, "\r")
			entry := models.ParseLogLine(line, w.stream)
			*w.entries = append(*w.entries, entry)
		}
	}

	return len(p), nil
}

func (w *logWriter) Flush() {
	if len(w.buffer) == 0 {
		return
	}
	line := strings.TrimSuffix(string(w.buffer), "\r")
	if line != "" {
		entry := models.ParseLogLine(line, w.stream)
		*w.entries = append(*w.entries, entry)
	}
	w.buffer = nil
}

type streamingLogWriter struct {
	stream     string
	buffer     []byte
	encoder    *json.Encoder
	encoderMu  *sync.Mutex
	pipeWriter *io.PipeWriter
}

func (w *streamingLogWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)

	for {
		idx := -1
		for i, b := range w.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}

		if idx == -1 {
			break
		}

		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]

		if line != "" {
			line = strings.TrimSuffix(line, "\r")
			if encodeErr := w.emit(line); encodeErr != nil {
				return 0, encodeErr
			}
		}
	}

	return len(p), nil
}

func (w *streamingLogWriter) Flush() {
	if len(w.buffer) == 0 {
		return
	}
	line := strings.TrimSuffix(string(w.buffer), "\r")
	if line != "" {
		_ = w.emit(line)
	}
	w.buffer = nil
}

func (w *streamingLogWriter) emit(line string) error {
	entry := models.ParseLogLine(line, w.stream)

	w.encoderMu.Lock()
	err := w.encoder.Encode(entry)
	w.encoderMu.Unlock()

	if err != nil {
		w.pipeWriter.CloseWithError(err)
	}

	return err
}

func buildLogsOptions(options models.LogOptions, follow, timestamps bool) container.LogsOptions {
	return container.LogsOptions{
		Follow:     follow,
		Timestamps: timestamps,
		Details:    options.Details,
		Since:      options.Since,
		Until:      options.Until,
		Tail:       options.Tail,
		ShowStdout: options.ShowStdout,
		ShowStderr: options.ShowStderr,
	}
}

// multi host client methods
func (c *MultiHostClient) GetContainerLogsParsed(hostName, id string, options models.LogOptions) ([]models.LogEntry, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	logs, err := apiClient.ContainerLogs(context.Background(), id, buildLogsOptions(options, false, true))
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	return parseDockerLogs(logs)
}

func (c *MultiHostClient) StreamContainerLogsParsed(hostName, id string, options models.LogOptions) (io.ReadCloser, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	logs, err := apiClient.ContainerLogs(context.Background(), id, buildLogsOptions(options, options.Follow, true))
	if err != nil {
		return nil, err
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer logs.Close()
		defer pipeWriter.Close()

		encoder := json.NewEncoder(pipeWriter)
		var mu sync.Mutex

		stdout := &streamingLogWriter{
			stream:     "stdout",
			encoder:    encoder,
			encoderMu:  &mu,
			pipeWriter: pipeWriter,
		}
		stderr := &streamingLogWriter{
			stream:     "stderr",
			encoder:    encoder,
			encoderMu:  &mu,
			pipeWriter: pipeWriter,
		}

		_, err = stdcopy.StdCopy(stdout, stderr, logs)
		stdout.Flush()
		stderr.Flush()

		if err != nil && err != io.EOF {
			pipeWriter.CloseWithError(err)
		}
	}()

	return pipeReader, nil
}

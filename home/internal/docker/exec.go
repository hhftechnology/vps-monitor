package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// CreateExec creates an exec instance for a container
// It tries to launch /bin/bash, falling back to /bin/sh if bash is not available
func (c *MultiHostClient) CreateExec(ctx context.Context, host, containerID string) (string, error) {
	cli, err := c.GetClient(host)
	if err != nil {
		return "", err
	}

	// We use a shell command that tries bash first, then sh
	// This is a common pattern to get the best available shell
	cmd := []string{"/bin/sh", "-c", "(test -x /bin/bash && exec /bin/bash) || exec /bin/sh"}

	config := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          cmd,
	}

	response, err := cli.ContainerExecCreate(ctx, containerID, config)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	return response.ID, nil
}

// AttachExec attaches to an existing exec instance and returns the hijacked response
func (c *MultiHostClient) AttachExec(ctx context.Context, host, execID string) (*types.HijackedResponse, error) {
	cli, err := c.GetClient(host)
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerExecAttach(ctx, execID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}

	return &resp, nil
}

// ResizeExec resizes the tty for an exec instance
func (c *MultiHostClient) ResizeExec(ctx context.Context, host, execID string, height, width uint) error {
	cli, err := c.GetClient(host)
	if err != nil {
		return err
	}

	return cli.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Height: height,
		Width:  width,
	})
}

// GetClientExport exposes GetClient for use in other packages if needed
// although it is already exported in client.go, this is just a comment reminder

